package worker

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var logger log.Logger

func init() {
	var err error
	logger, err = log.New("worker-test", log.DebugLevel)
	if err != nil {
		panic(err)
	}
}

const startNodeID = "n1"

func GetWork(id int) work.Work {
	if id <= 0 {
		panic("id must be positive")
	}

	s := strconv.Itoa(id)
	return work.Work{
		ID:                 s,
		WorkflowID:         "w-" + s,
		StartNodeID:        startNodeID,
		StartNodePayload:   []byte(fmt.Sprintf(`{"a": %s}`, s)),
		MaxDurationSeconds: id,
		MaxSteps:           id,
	}
}

func GetWorkflow(id string, workflowStatus model.WorkflowStatus) model.WorkflowWithNodesCredential {
	return model.WorkflowWithNodesCredential{
		Workflow: model.Workflow{
			ID:          "workflow-id-" + id,
			Name:        "workflow-name-" + id,
			Status:      workflowStatus,
			Description: "a workflow that does not work at all.",
		},
		Nodes: model.NodesWithCredential{
			{
				Node: model.Node{
					ID: startNodeID,
					Data: model.NodeData{
						MetaData: model.NodeMetaData{
							EnableTriggerAtFirst: true,
						},
					},
				},
			},
		},
	}
}

var errBadLuck = errors.New("bad luck")

func GetExecutionResult(workID string) workflow.ExecutionResult {
	var (
		hash       = sha256.Sum256([]byte(workID))
		err        error
		status     model.WorkflowInstanceStatus
		failNodeID string
	)

	if hash[0] > 127 {
		status = model.WorkflowInstanceStatusCompleted
	} else {
		status = model.WorkflowInstanceStatusFailed
		err = errBadLuck
		failNodeID = "node-" + workID
	}

	return workflow.ExecutionResult{
		Status:     status,
		FailNodeID: failNodeID,
		Err:        err,
		InstanceNodes: []*model.WorkflowInstanceNode{
			{
				WorkflowInstanceID: workID,
			},
		},
	}
}

type DummyWorkSource struct {
	autoProduct         bool
	eof                 bool
	consumerMaxCapacity int
	cursor              int
	queue               chan int
}

func NewDummyWorkSource(autoProduct bool, EOF bool, consumerMaxCapacity int) *DummyWorkSource {
	return &DummyWorkSource{
		autoProduct:         autoProduct,
		eof:                 EOF,
		consumerMaxCapacity: consumerMaxCapacity,
		cursor:              0,
		queue:               make(chan int, consumerMaxCapacity),
	}
}

func (d *DummyWorkSource) Product() {
	d.cursor++

	if d.cursor > d.consumerMaxCapacity {
		panic("cursor is too large")
	}

	d.queue <- d.cursor
}

func (d *DummyWorkSource) Consume(ctx context.Context) (newWork work.Work, err error) {
	if d.eof {
		return work.Work{}, io.EOF
	}

	if d.autoProduct {
		if d.cursor < d.consumerMaxCapacity {
			d.cursor++
			return GetWork(d.cursor), nil
		}

		<-ctx.Done()
		return work.Work{}, ctx.Err()
	}

	select {
	case <-ctx.Done():
		return work.Work{}, ctx.Err()
	case id := <-d.queue:
		return GetWork(id), nil
	}
}

type DummyWorkflowStore struct {
	mu             sync.Mutex
	workflowStatus model.WorkflowStatus
	// key: id of the WorkflowInstance
	M       map[string]model.WorkflowExecutingUpdatePayload
	Samples []*model.WorkflowInstanceNode
}

func NewDummyWorkflowStore(workflowStatus model.WorkflowStatus) *DummyWorkflowStore {
	return &DummyWorkflowStore{
		workflowStatus: workflowStatus,
		M:              make(map[string]model.WorkflowExecutingUpdatePayload),
	}
}

func (d *DummyWorkflowStore) GetWorkflowWithNodesCredentialByID(ctx context.Context, workflowID string) (workflowWithNodes model.WorkflowWithNodesCredential, err error) {
	return GetWorkflow(workflowID, d.workflowStatus), nil
}

func (d *DummyWorkflowStore) FinishWorkflowInstanceExecution(ctx context.Context, result model.WorkflowExecutingUpdatePayload) (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if result.ID == "" {
		err = errors.New("result ID is empty")
		return
	}

	_, ok := d.M[result.ID]
	if ok {
		err = fmt.Errorf("record already exists, id: %s", result.ID)
		return
	}

	d.M[result.ID] = result
	return
}

func (d *DummyWorkflowStore) InsertWorkflowInstanceNode(ctx context.Context, record *model.WorkflowInstanceNode) (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.Samples = append(d.Samples, record)

	return
}

func (d *DummyWorkflowStore) DeleteWorkflowInstanceByID(ctx context.Context, id string) (err error) {
	return nil
}

func TestWorker_Run_Shutdown(t *testing.T) {
	const concurrency = 5

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert := require.New(t)
	worker, err := newWorker(option{
		Option: Option{
			WorkSource: NewDummyWorkSource(false, false, 8),
			Log:        logger,
			Config: Config{
				GlobalMaxSteps:           5,
				GlobalMaxDurationSeconds: 5,
				Concurrency:              concurrency,
			},
		},
	})
	assert.NoError(err)

	assert.Zero(worker.onlineProcessorNum.Load())
	assert.Zero(worker.onlineIntakeNum.Load())

	workerHasQuit := make(chan struct{})

	go func() {
		runErr := worker.Run()
		assert.NoError(runErr)

		close(workerHasQuit)
	}()

	// wait for scheduler
	time.Sleep(1 * time.Second)

	assert.Equal(int32(concurrency), worker.onlineProcessorNum.Load())
	assert.Equal(int32(1), worker.onlineIntakeNum.Load())

	err = worker.Shutdown(ctx)
	assert.NoError(err)

	select {
	case <-workerHasQuit:
	// relax
	case <-ctx.Done():
		err = fmt.Errorf("waiting for shutdown: %w", ctx.Err())
		assert.NoError(err)
	}

	assert.Zero(worker.onlineProcessorNum.Load())
	assert.Zero(worker.onlineIntakeNum.Load())
}

func TestWorker_intake_CriticalError(t *testing.T) {
	const concurrency = 5

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert := require.New(t)
	worker, err := newWorker(option{
		Option: Option{
			WorkSource: NewDummyWorkSource(false, true, 8),
			Log:        logger,
			Config: Config{
				GlobalMaxSteps:           5,
				GlobalMaxDurationSeconds: 5,
				Concurrency:              concurrency,
			},
		},
	})
	assert.NoError(err)

	workerHasQuit := make(chan struct{})

	// prevent deadlock
	go func() {
		runErr := worker.Run()
		assert.Error(runErr)
		assert.True(errors.Is(runErr, io.EOF))

		close(workerHasQuit)
	}()

	select {
	case <-workerHasQuit:
	// relax
	case <-ctx.Done():
		err = fmt.Errorf("waiting for Worker to report error: %w", ctx.Err())
		assert.NoError(err)
	}
}

// TestWorker_PoolingEdgeCase To create the edge case,
// we inject a workflow executing callback with relative long executing time into the processor.
// As long as there's only one processor online,
// we can always reliably(to some extent) reproduce the edge case
// if we shut down the runner after we "feed" the processor in the right timing.
func TestWorker_PoolingEdgeCase(t *testing.T) {
	const concurrency = 1

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	store := NewDummyWorkflowStore(model.WorkflowStatusEnabled)
	workSource := NewDummyWorkSource(false, false, 8)

	assert := require.New(t)
	worker, err := newWorker(option{
		Option: Option{
			WorkSource: workSource,
			Log:        logger,
			Config: Config{
				GlobalMaxSteps:           5,
				GlobalMaxDurationSeconds: 5,
				Concurrency:              concurrency,
			},
		},
		WorkflowStore: store,
		WorkflowTestExecutingCallback: func(ctx context.Context, newWork work.Work, workflowCtxOpt workflow.ContextOpt) (result workflow.ExecutionResult) {
			time.Sleep(2 * time.Second)
			result.Status = model.WorkflowInstanceStatusCompleted
			return
		},
	})
	assert.NoError(err)

	assert.Zero(worker.onlineProcessorNum.Load())
	assert.Zero(worker.onlineIntakeNum.Load())

	workerHasQuit := make(chan struct{})

	go func() {
		runErr := worker.Run()
		assert.NoError(runErr)

		close(workerHasQuit)
	}()

	// wait for scheduler
	time.Sleep(1 * time.Second)

	assert.Equal(int32(concurrency), worker.onlineProcessorNum.Load())
	assert.Equal(int32(1), worker.onlineIntakeNum.Load())

	// feed the worker with three pieces of work in a roll
	workSource.Product()
	workSource.Product()
	workSource.Product()

	// wait for scheduler
	time.Sleep(1 * time.Second)

	// now the first work is running
	err = worker.Shutdown(ctx)
	assert.NoError(err)

	select {
	case <-workerHasQuit:
	// relax
	case <-ctx.Done():
		err = fmt.Errorf("waiting for shutdown: %w", ctx.Err())
		assert.NoError(err)
	}

	assert.Zero(worker.onlineProcessorNum.Load())
	assert.Zero(worker.onlineIntakeNum.Load())

	// ensure the second work has also run.
	assert.Len(store.M, 2)
	for k, v := range store.M {
		assert.Equal(model.WorkflowInstanceStatusCompleted, v.Status, k)
	}
}

// TestWorker_RunningNormallyWhenWorkflowDisable data will write into samples if workflow is disabled.
func TestWorker_RunningNormallyWhenWorkflowDisable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const (
		concurrency = 3
		workNum     = 11
	)

	store := NewDummyWorkflowStore(model.WorkflowStatusDisabled)
	workSource := NewDummyWorkSource(true, false, workNum)

	assert := require.New(t)
	worker, err := newWorker(option{
		Option: Option{
			WorkSource: workSource,
			Log:        logger,
			Config: Config{
				GlobalMaxSteps:           5,
				GlobalMaxDurationSeconds: 5,
				Concurrency:              concurrency,
			},
		},
		WorkflowStore: store,
	})
	worker.runTriggerNodeForSample = func(ctx context.Context, w *Worker, workflowWithNodes model.WorkflowWithNodesCredential, node model.NodeWithCredential, newWork work.Work) (err error) {
		return w.workflowStore.InsertWorkflowInstanceNode(ctx, &model.WorkflowInstanceNode{
			WorkflowID: workflowWithNodes.ID,
		})
	}
	assert.NoError(err)

	workerHasQuit := make(chan struct{})

	go func() {
		runErr := worker.Run()
		assert.NoError(runErr)

		close(workerHasQuit)
	}()

	// wait for scheduler
	time.Sleep(1 * time.Second)

	err = worker.Shutdown(ctx)
	assert.NoError(err)

	assert.Len(store.Samples, workNum)
}

func TestWorker_RunningNormally(t *testing.T) {
	const (
		concurrency = 3
		workNum     = 11
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := NewDummyWorkflowStore(model.WorkflowStatusEnabled)
	workSource := NewDummyWorkSource(true, false, workNum)

	assert := require.New(t)
	worker, err := newWorker(option{
		Option: Option{
			WorkSource: workSource,
			Log:        logger,
			Config: Config{
				GlobalMaxSteps:           5,
				GlobalMaxDurationSeconds: 5,
				Concurrency:              concurrency,
			},
		},
		WorkflowStore: store,
		WorkflowTestExecutingCallback: func(ctx context.Context, newWork work.Work, workflowCtxOpt workflow.ContextOpt) (result workflow.ExecutionResult) {
			var err error

			workID, err := strconv.Atoi(newWork.ID)
			assert.NoError(err)
			assert.Equal(GetWork(workID), newWork)
			assert.Equal(newWork.ID, workflowCtxOpt.WorkflowInstanceID)
			assert.Equal(GetWorkflow(newWork.WorkflowID, model.WorkflowStatusEnabled), workflowCtxOpt.WorkflowWithNodes)

			result = GetExecutionResult(newWork.ID)
			err = ctx.Err()
			if err != nil {
				result.Err = err
			}

			return
		},
	})
	assert.NoError(err)

	assert.Zero(worker.onlineProcessorNum.Load())
	assert.Zero(worker.onlineIntakeNum.Load())

	workerHasQuit := make(chan struct{})

	go func() {
		runErr := worker.Run()
		assert.NoError(runErr)

		close(workerHasQuit)
	}()

	// wait for scheduler
	time.Sleep(1 * time.Second)

	assert.Equal(int32(concurrency), worker.onlineProcessorNum.Load())
	assert.Equal(int32(1), worker.onlineIntakeNum.Load())

	// wait for scheduler
	time.Sleep(2 * time.Second)

	err = worker.Shutdown(ctx)
	assert.NoError(err)

	select {
	case <-workerHasQuit:
	// relax
	case <-ctx.Done():
		err = fmt.Errorf("waiting for shutdown: %w", ctx.Err())
		assert.NoError(err)
	}

	assert.Zero(worker.onlineProcessorNum.Load())
	assert.Zero(worker.onlineIntakeNum.Load())

	assert.Len(store.M, workNum)
	for k, v := range store.M {
		expectedResult := GetExecutionResult(k)

		assert.Equal(expectedResult.Status, v.Status, k)
		assert.True(errors.Is(v.Err, expectedResult.Err), k)
		assert.Equal(expectedResult.InstanceNodes, v.InstanceNodes, k)
		assert.Equal(expectedResult.FailNodeID, v.FailNodeID, k)
	}
}

func TestWorker_collapsedMaxSteps(t *testing.T) {
	tests := []struct {
		name           string
		globalMaxSteps int
		workMaxSteps   int
		want           int
	}{
		{
			name:           "local is 0",
			globalMaxSteps: 5,
			workMaxSteps:   0,
			want:           5,
		},
		{
			name:           "global is big enough",
			globalMaxSteps: 5,
			workMaxSteps:   1,
			want:           1,
		},
		{
			name:           "global equals local",
			globalMaxSteps: 5,
			workMaxSteps:   5,
			want:           5,
		},
		{
			name:           "global is less than local",
			globalMaxSteps: 5,
			workMaxSteps:   10,
			want:           5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{
				globalMaxSteps: tt.globalMaxSteps,
			}
			if got := w.collapsedMaxSteps(tt.workMaxSteps); got != tt.want {
				t.Errorf("collapsedMaxSteps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorker_collapsedMaxDuration(t *testing.T) {
	tests := []struct {
		name                   string
		globalMaxDuration      time.Duration
		workMaxDurationSeconds int
		want                   time.Duration
	}{
		{
			name:                   "local is 0",
			globalMaxDuration:      5 * time.Second,
			workMaxDurationSeconds: 0,
			want:                   5 * time.Second,
		},
		{
			name:                   "global is big enough",
			globalMaxDuration:      5 * time.Second,
			workMaxDurationSeconds: 3,
			want:                   3 * time.Second,
		},
		{
			name:                   "global equals local",
			globalMaxDuration:      5 * time.Second,
			workMaxDurationSeconds: 5,
			want:                   5 * time.Second,
		},
		{
			name:                   "global is less than local",
			globalMaxDuration:      5 * time.Second,
			workMaxDurationSeconds: 8,
			want:                   5 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{
				globalMaxDuration: tt.globalMaxDuration,
			}
			if got := w.collapsedMaxDuration(tt.workMaxDurationSeconds); got != tt.want {
				t.Errorf("collapsedMaxDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
