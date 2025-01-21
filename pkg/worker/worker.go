package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"

	"go.uber.org/atomic"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"go.uber.org/zap"

	"github.com/getsentry/sentry-go"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
)

type Config struct {
	// how many workflow can be executed at the same time?
	// try usableCores * 40 as a start point
	Concurrency int `comment:"how many workflow can be executed at the same time? try usableCores * 40 as a start point"`
	// how many steps can a workflow executes at most globally?
	// must be positive
	GlobalMaxSteps int `comment:"must be positive, how many steps can a workflow executes at most globally?"`
	// global workflow execution timeout.
	// must be positive
	GlobalMaxDurationSeconds int `comment:"must be positive, global workflow execution timeout"`

	// Where pending work comes, typically consumed by Worker
	//
	// WorkSource are actually initialized outside Worker's factory.
	// We put this field here for conceptional completeness.
	WorkSource work.ConsumerConfig `comment:"Where pending workflow execution(aka, work) comes"`
}

// WorkflowStore is used by Worker for workflow querying and execution result keeping
type WorkflowStore interface {
	GetWorkflowWithNodesCredentialByID(ctx context.Context, workflowID string) (workflowWithNodes model.WorkflowWithNodesCredential, err error)
	FinishWorkflowInstanceExecution(ctx context.Context, payload model.WorkflowExecutingUpdatePayload) (err error)
	InsertWorkflowInstanceNode(ctx context.Context, record *model.WorkflowInstanceNode) (err error)
	DeleteWorkflowInstanceByID(ctx context.Context, id string) (err error)
}

type workflowStore struct {
	*model.DB
}

func (w workflowStore) FinishWorkflowInstanceExecution(ctx context.Context, payload model.WorkflowExecutingUpdatePayload) (err error) {
	err = w.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.UpdateExecutingResultIntoWorkflowInstance(ctx, payload)
		if err != nil {
			err = fmt.Errorf("update workflow execution: %w", err)
			return
		}

		err = tx.BulkInsertWorkflowInstanceNodes(ctx, payload.InstanceNodes)
		if err != nil {
			err = fmt.Errorf("bulk insert workflow instance nodes: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("run in tx: %w", err)
		return
	}
	return
}

type WorkSource interface {
	Consume(ctx context.Context) (newWork work.Work, err error)
}

type workflowTestExecutingCallback func(ctx context.Context, newWork work.Work, workflowCtxOpt workflow.ContextOpt) (result workflow.ExecutionResult)

type runTriggerNodeForSample func(ctx context.Context,
	w *Worker,
	workflowWithNodes model.WorkflowWithNodesCredential,
	node model.NodeWithCredential,
	newWork work.Work) (err error)

// Worker orchestras work intake and multiple processors, executes incoming works.
//
// A worker can not be reused once Run is called, even after Shutdown is called.
type Worker struct {
	// where to talk
	log log.Logger
	// where work comes from
	workSource WorkSource
	// where to query workflow and keep execution result
	// normally it's *model.DB, the same value of db field.
	// here we use different fields to make unit test feasible.
	workflowStore WorkflowStore

	// dependency of workflow.WorkflowContext
	db *model.DB
	// dependency of workflow.WorkflowContext
	cache *cache.Cache
	// dependency of workflow.WorkflowContext
	cipher crypto.CryptoCipher
	// dependency of workflow.WorkflowContext
	serverHost *serverhost.ServerHost
	// dependency of workflow.WorkflowContext
	mailSender *smtp.Sender
	// dependency of workflow.WorkflowContext
	passportVendorLookup map[model.PassportVendorName]model.PassportVendor

	runTriggerNodeForSample runTriggerNodeForSample

	// for testing purpose
	workflowTestExecutingCallback workflowTestExecutingCallback

	// closed when Shutdown is called
	shuttingDown chan struct{}
	// closed by Shutdown to inform Run to exit
	shutdownFinished chan struct{}
	// listened by Run and reported by intake and process, buffered
	criticalErr chan error

	// the mother of all contexts of running works
	processCtx context.Context
	// called by Shutdown to cancel processCtx
	cancelProcessCtx context.CancelFunc

	// incoming works, unbuffered
	intaker chan work.Work
	// used by workSource in intake
	intakerCtx context.Context
	// called by Shutdown to cancel intakerCtx
	cancelIntakeCtx context.CancelFunc

	// how many steps can a workflow executes at most globally?
	// must be positive
	globalMaxSteps int
	// global workflow execution timeout.
	// must be positive
	globalMaxDuration time.Duration
	// how many workflow can be executed at the same time?
	concurrency int

	// how many processors are online?
	onlineProcessorNum atomic.Int32
	// how many intakes are online?
	onlineIntakeNum atomic.Int32
}

type Option struct {
	Log                  log.Logger
	DB                   *model.DB
	Cache                *cache.Cache
	WorkSource           WorkSource
	Cipher               crypto.CryptoCipher
	ServerHost           *serverhost.ServerHost
	MailSender           *smtp.Sender
	PassportVendorLookup map[model.PassportVendorName]model.PassportVendor

	Config
}

func newWorkflowStore(db *model.DB) WorkflowStore {
	return &workflowStore{
		DB: db,
	}
}

type option struct {
	Option

	WorkflowStore                 WorkflowStore
	WorkflowTestExecutingCallback workflowTestExecutingCallback
}

// newWorker decouples WorkflowStore and model.DB to make unit test feasible.
func newWorker(opt option) (worker *Worker, err error) {
	if opt.Concurrency < 1 {
		err = fmt.Errorf("worker concurrency must be at least 1, got %d", opt.Concurrency)
		return
	}
	if maxMeaningfulConcurrency := runtime.NumCPU() * 100; opt.Concurrency > maxMeaningfulConcurrency {
		err = fmt.Errorf("worker concurrency is meaninglessly high, got %d, want something below %d", opt.Concurrency, maxMeaningfulConcurrency)
		return
	}

	if opt.GlobalMaxSteps <= 0 {
		err = fmt.Errorf("invalid globalMaxSteps, want it positive, got %d", opt.GlobalMaxSteps)
		return
	}
	if opt.GlobalMaxDurationSeconds <= 0 {
		err = fmt.Errorf("invalid globalMaxDurationMs, want it positive, got %d", opt.GlobalMaxSteps)
		return
	}

	processCtx, cancelProcessCtx := context.WithCancel(context.Background())
	intakerCtx, cancelIntakeCtx := context.WithCancel(context.Background())

	worker = &Worker{
		log:                           opt.Log,
		workSource:                    opt.WorkSource,
		workflowStore:                 opt.WorkflowStore,
		db:                            opt.DB,
		cache:                         opt.Cache,
		cipher:                        opt.Cipher,
		serverHost:                    opt.ServerHost,
		mailSender:                    opt.MailSender,
		passportVendorLookup:          opt.PassportVendorLookup,
		workflowTestExecutingCallback: opt.WorkflowTestExecutingCallback,
		shuttingDown:                  make(chan struct{}),
		shutdownFinished:              make(chan struct{}),
		criticalErr:                   make(chan error, 1), // never blocks critical error sender
		processCtx:                    processCtx,
		cancelProcessCtx:              cancelProcessCtx,
		intaker:                       make(chan work.Work),
		intakerCtx:                    intakerCtx,
		cancelIntakeCtx:               cancelIntakeCtx,
		globalMaxSteps:                opt.GlobalMaxSteps,
		globalMaxDuration:             time.Duration(opt.GlobalMaxDurationSeconds) * time.Second,
		concurrency:                   opt.Concurrency,
		onlineProcessorNum:            atomic.Int32{},
		onlineIntakeNum:               atomic.Int32{},
	}

	return
}

// NewWorker factory of Worker.
func NewWorker(opt Option) (worker *Worker, err error) {
	return newWorker(option{
		Option:                        opt,
		WorkflowStore:                 newWorkflowStore(opt.DB),
		WorkflowTestExecutingCallback: nil,
	})
}

// Run start the Worker and blocks, until one of the following situations which comes first:
//
// * Worker encounters unrecoverable errors, like the workSource is not available anymore;
// * Shutdown is called and FINISHED the shutdown process, either gracefully or not.
func (w *Worker) Run() (err error) {
	// start process firstly, then intake
	for i := 0; i < w.concurrency; i++ {
		go w.process(i)
	}

	// watch for new works
	go w.intake()

	w.log.Info("worker has started.", zap.Int("concurrency", w.concurrency))
	defer w.log.Info("worker is down.")

	// blocks and wait for shutdown
	select {
	case err = <-w.criticalErr:
		err = fmt.Errorf("critical error: %w", err)
		return
	case <-w.shutdownFinished:
		// relax
		return
	}
}

// Shutdown tries to stop the Worker gracefully given time allowance by ctx.
// During graceful shutdown, Worker stops accepting new works and existed works continue running.
// If ctx is canceled, a force shutdown will be triggered and contexts that are provided
// to existed works are canceled.
func (w *Worker) Shutdown(ctx context.Context) (err error) {
	// stop receiving new work first, before stop processors
	close(w.shuttingDown)
	w.cancelIntakeCtx()

	done := ctx.Done()
	timer := time.NewTicker(200 * time.Millisecond)
	defer timer.Stop()
	defer close(w.shutdownFinished)

	for {
		select {
		case <-done:
			// force processor to stop
			w.cancelProcessCtx()
			err = ctx.Err()
			return
		case <-timer.C:
			if w.onlineIntakeNum.Load() == 0 && w.onlineProcessorNum.Load() == 0 {
				return
			}

			// If at first you don't succeed, try, try again.
			continue
		}
	}
}

// intake runs in a goroutine and deliver incoming work into intaker channel.
func (w *Worker) intake() {
	w.onlineIntakeNum.Add(1)
	defer func() {
		w.onlineIntakeNum.Add(-1)
		w.log.Debug("intake is down.")
	}()
	w.log.Debug("intake is up.")

	for {
		select {
		case <-w.shuttingDown:
			// shutting down, let's exit
			w.log.Info("intake got shutting down signal, exit.")
			return
		default:
			// relax and carry on
		}

		newWork, err := w.workSource.Consume(w.intakerCtx)
		if errors.Is(err, io.EOF) ||
			errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			// are we exiting?
			select {
			case <-w.shuttingDown:
				// nothing is on fire
				err = nil
				return
			default:
				// continue following logics
			}

			// workSource is closed, which seldom happens but must be regard as a critical error
			err = fmt.Errorf("workSource meets critical error during work consuming: %w", err)
			w.log.Error("intake error", zap.Error(err))

			// Don't block. There's always one winner since it's a one-buffered channel.
			select {
			case w.criticalErr <- err:
				// relax
			default:
				// relax
			}

			return
		}
		if err != nil {
			// notable err, report to Sentry
			err = fmt.Errorf("consuming work: %w", err)
			w.log.Error("intake error", zap.Error(err))
			event := &sentry.Event{
				Level:   sentry.LevelError,
				Message: err.Error(),
			}
			sentry.CaptureEvent(event)

			continue
		}

		// deliver to process
		w.log.Info("intake got new work",
			zap.String("workId", newWork.ID),
			zap.String("workflowId", newWork.WorkflowID),
		)
		w.intaker <- newWork
	}
}

// process runs in a goroutine fungibly and executes the work from intaker
// There are concurrent(numbers of) process goroutines running at the same time.
func (w *Worker) process(processorID int) {
	w.onlineProcessorNum.Add(1)
	defer func() {
		w.onlineProcessorNum.Add(-1)
		w.log.Debug("processor is down.", zap.Int("processorId", processorID))
	}()

	w.log.Debug("processor is up.", zap.Int("processorId", processorID))

	for {
		var newWork work.Work

	POLLING:
		select {
		case <-w.shuttingDown:
			select {
			case newWork = <-w.intaker:
				// Edge case: During shutting down, there may be one piece of work left in the channel,
				// and we must still take care of it.
				break POLLING
			default:
				w.log.Debug("processor got shutting down signal, exiting...", zap.Int("processorId", processorID))
				return
			}
		case newWork = <-w.intaker:
			// carry on the following logics
		}

		w.log.Info("processor got new work",
			zap.String("workId", newWork.ID),
			zap.String("workflowId", newWork.WorkflowID),
			zap.Int("processorId", processorID),
		)

		startedAt := time.Now()
		err := w.processOne(processorID, newWork)
		if err != nil {
			err = fmt.Errorf("processor excuting work: %w", err)

			event := &sentry.Event{
				Level:   sentry.LevelInfo,
				Message: err.Error(),
				Tags: map[string]string{
					"workId":      newWork.ID,
					"workflowId":  newWork.WorkflowID,
					"processorId": strconv.Itoa(processorID),
				},
			}
			sentry.CaptureEvent(event)

			w.log.Warn("processor failed to execute work",
				zap.String("workId", newWork.ID),
				zap.String("workflowId", newWork.WorkflowID),
				zap.Int("processorId", processorID),
				zap.Duration("timeTaken", time.Since(startedAt)),
				zap.Error(err),
			)

			continue
		}

		w.log.Info("processor successfully executed work",
			zap.String("workId", newWork.ID),
			zap.String("workflowId", newWork.WorkflowID),
			zap.Int("processorId", processorID),
			zap.Duration("timeTaken", time.Since(startedAt)),
		)
	}
}

func (w *Worker) processOne(processorID int, newWork work.Work) (err error) {
	ctx, cancel := context.WithTimeout(w.processCtx, w.collapsedMaxDuration(newWork.MaxDurationSeconds))
	defer cancel()

	// Always put the following two defers at the start of the function.
	// This ensures the status of workflow instance gets updated
	// whenever the execution succeeds or fails.
	var (
		result          workflow.ExecutionResult
		skipBookkeeping bool
	)
	defer func() {
		payload := model.WorkflowExecutingUpdatePayload{
			ID:            newWork.ID,
			WantStatus:    model.WorkflowInstanceStatusRunning,
			Status:        result.Status,
			DurationMs:    int(result.Duration.Milliseconds()),
			Steps:         result.Steps,
			FailNodeID:    result.FailNodeID,
			Err:           result.Err,
			InstanceNodes: result.InstanceNodes,
		}
		if errors.Is(err, workflow.ErrNeedWorkflowPaused) {
			payload.Status = model.WorkflowInstanceStatusPaused
			// silence the error
			err = nil
		} else if err != nil {
			payload.Status = model.WorkflowInstanceStatusFailed
			payload.Err = err
		}

		if skipBookkeeping {
			return
		}

		bookkeepingCtx, cancelBookkeeping := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelBookkeeping()

		badLuck := w.workflowStore.FinishWorkflowInstanceExecution(bookkeepingCtx, payload)
		if badLuck == nil {
			return
		}

		w.log.Warn("failed to update executing result into workflow instance", zap.Error(err))
		if err == nil {
			err = fmt.Errorf("workflow executed successfully, but update workflow instance failed: %w", badLuck)
			return
		}
	}()
	defer func() {
		recovered := recover()
		if recovered == nil {
			// relax
			return
		}

		stack := debug.Stack()

		switch typed := recovered.(type) {
		case string:
			err = fmt.Errorf("panic recovered: %s, original stack: %s", typed, stack)
		case error:
			err = fmt.Errorf("panic recovered: %w, original stack: %s", typed, stack)
		default:
			err = fmt.Errorf("panic recovered: %#v, original stack: %s", typed, stack)
		}
	}()

	workflowWithNodes, err := w.workflowStore.GetWorkflowWithNodesCredentialByID(w.processCtx, newWork.WorkflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow with nodes: %w", err)
		return
	}

	node, ok := workflowWithNodes.Nodes.MapByID()[newWork.StartNodeID]
	if !ok {
		w.log.Error("workflow node not found",
			zap.String("workId", newWork.ID),
			zap.String("workflowId", newWork.WorkflowID),
			zap.String("nodeID", node.ID))
		return
	}

	if workflowWithNodes.Status != model.WorkflowStatusEnabled {
		// Here is a nasty hack,
		// defaultRunTriggerNodeForSample deletes the corresponding workflow instance record,
		// so we can not update its status in the defer func.
		skipBookkeeping = true

		// write the request to sample of this node.
		fn := w.runTriggerNodeForSample
		if fn == nil {
			fn = defaultRunTriggerNodeForSample
		}
		err = fn(ctx, w, workflowWithNodes, node, newWork)
		if err != nil {
			err = fmt.Errorf("run first node failed: %v", err)
			return
		}

		// consider the work is done.
		return
	}

	// TODO(sword): enable validation.
	// frontend bug causes some nodes' transition not found.
	// report := validate.ValidateWorkflow(workflowWithNodes)
	// if report.ExistsFatal() {
	// 	err = fmt.Errorf("validating workflow: %s", report.Err())
	// 	return
	// }

	workflowCtxOpt := workflow.ContextOpt{
		DB:                   w.db,
		Cache:                w.cache,
		WorkflowWithNodes:    workflowWithNodes,
		Cipher:               w.cipher,
		ServerHost:           w.serverHost,
		MaxSteps:             w.collapsedMaxSteps(newWork.MaxSteps),
		WorkflowInstanceID:   newWork.ID,
		MailSender:           w.mailSender,
		PassportVendorLookup: w.passportVendorLookup,
	}

	// testing hook
	if w.workflowTestExecutingCallback == nil {
		var workflowCtx *workflow.WorkflowContext

		if newWork.Resume {
			workflowCtx, err = workflow.NewResumingWorkflowContext(ctx, workflowCtxOpt)
			if err != nil {
				err = fmt.Errorf("building resuming workflow context: %w", err)
				return
			}
		} else {
			workflowCtx, err = workflow.NewWorkflowContext(ctx, workflowCtxOpt)
			if err != nil {
				err = fmt.Errorf("building new workflow context: %w", err)
				return
			}
		}

		result = workflowCtx.Run(newWork.StartNodeID, newWork.StartNodePayload)
	} else {
		result = w.workflowTestExecutingCallback(ctx, newWork, workflowCtxOpt)
	}

	// don't forget to grab the error
	err = result.Err
	if err != nil {
		err = fmt.Errorf("executing workflow: %w", err)
		return
	}

	return
}

// defaultRunTriggerNodeForSample, run workflow trigger node, then write the output to samples.
func defaultRunTriggerNodeForSample(ctx context.Context, w *Worker, workflowWithNodes model.WorkflowWithNodesCredential, node model.NodeWithCredential, newWork work.Work) (err error) {
	// if this work for sample, delete the workflow instance.
	// we will refactor this in the future, read payload from the mq, so don't need to delete workflow instance here.
	defer func() {
		err = w.workflowStore.DeleteWorkflowInstanceByID(ctx, newWork.ID)
		if err != nil {
			err = fmt.Errorf("delete workflow instance: %w", err)
			return
		}
	}()

	opt := workflow.TestWorkflowActionOpt{
		BaseWorkflowActionOpt: workflow.BaseWorkflowActionOpt{
			Ctx:                  ctx,
			WorkflowWithNodes:    workflowWithNodes,
			DB:                   w.db,
			Cache:                w.cache,
			PassportVendorLookup: w.passportVendorLookup,
			MailSender:           w.mailSender,
			Cipher:               w.cipher,
			ServerHost:           w.serverHost,
		},
		NodeID: node.ID,
	}

	testAction, err := workflow.NewTestWorkflowAction(opt)
	if err != nil {
		err = fmt.Errorf("init TestWorkflowAction: %w", err)
		return
	}
	err = testAction.RunStartNode(newWork.StartNodePayload)
	if err != nil {
		w.log.For(ctx).Error("run trigger node failed",
			log.ErrField(err),
			zap.String("workId", newWork.ID),
			log.String("workflowID", newWork.WorkflowID),
			log.String("nodeID", newWork.StartNodeID))
		return
	}
	output := testAction.GetWorkflowContext().LookupScopeNodeData(newWork.StartNodeID)
	outputBytes, _ := json.Marshal(output)

	sampleVersion, _ := utils.ShortNanoID()
	err = w.workflowStore.InsertWorkflowInstanceNode(ctx, &model.WorkflowInstanceNode{
		WorkflowID:       workflowWithNodes.ID,
		NodeID:           node.ID,
		Status:           model.WorkflowInstanceNodeStatusCompleted,
		Class:            node.Class,
		Input:            newWork.StartNodePayload,
		Output:           outputBytes,
		StartTime:        time.Now(),
		Source:           model.NodeSourceLive,
		IsSelectedSample: false,
		IsSample:         false, // is_sample will be transformed to true when user click load more samples.
		SampleResourceID: node.ID,
		SampledAt:        time.Now(),
		SampleVersion:    sampleVersion,
	})
	if err != nil {
		err = fmt.Errorf("insert workflow instance node: %w", err)
		return
	}
	return
}

func (w *Worker) collapsedMaxSteps(workMaxSteps int) int {
	if workMaxSteps == 0 {
		return w.globalMaxSteps
	}

	if workMaxSteps > w.globalMaxSteps {
		return w.globalMaxSteps
	}

	return workMaxSteps
}

func (w *Worker) collapsedMaxDuration(workMaxDurationSeconds int) time.Duration {
	if workMaxDurationSeconds == 0 {
		return w.globalMaxDuration
	}

	workMaxDuration := time.Duration(workMaxDurationSeconds) * time.Second
	if workMaxDuration > w.globalMaxDuration {
		return w.globalMaxDuration
	}

	return workMaxDuration
}
