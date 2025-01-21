package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/schema"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/template"
)

// WorkflowContext One workflow run has a unique context.
type WorkflowContext struct {
	db               *model.DB
	cache            *cache.Cache
	cipher           crypto.CryptoCipher
	serverHost       *serverhost.ServerHost
	flowType         flowType
	startNodeName    string
	useExternalInput bool
	initialInput     any
	// nextNodeName is previous transition
	nextNodeName string
	currentNode  *nodeInstance
	status       model.WorkflowInstanceStatus
	stepNums     int
	maxStepNums  int
	// lostControl set when control is taken over by the logic node
	lostControl        bool
	logger             log.Logger
	context            context.Context
	workflow           model.WorkflowWithNodesCredential
	nodeLookup         map[string]*nodeInstance
	scope              contextScope
	workflowInstanceID string
	subflowContext     map[string][]*WorkflowContext
	// executionMode subflow context will inherit this.
	executionMode        executionMode
	mailSender           *smtp.Sender
	passportVendorLookup map[model.PassportVendorName]model.PassportVendor
}

type executionMode uint

const (
	executionNormalMode executionMode = iota
	executionTestMode                 // not persist workflow instance.
)

type flowType uint

const (
	workflowType flowType = iota
	subflowType
)

const defaultMaxSteps = 100

// NewTestWorkflowContext for test run node, so the node input from the context scope data.
// data value is node output.
func NewTestWorkflowContext(ctx context.Context, opt ContextOpt, data map[string]any) (workflowCtx *WorkflowContext, err error) {
	workflowCtx, err = NewWorkflowContext(ctx, opt)
	if err != nil {
		err = fmt.Errorf("building workflow context: %w", err)
		return
	}

	workflowCtx.useExternalInput = false
	for nodeID, output := range data {
		if nodeID == opt.testForeachNodeID {
			// if testing node in foreach, set the foreach output as Iter data.
			workflowCtx.scope.setIterData(output.(map[string]any))
		} else {
			workflowCtx.scope.setNodeData(nodeID, map[string]any{
				"output": output,
			})
		}
	}
	workflowCtx.setTestMode()
	return
}

type ContextOpt struct {
	DB                   *model.DB
	Cache                *cache.Cache
	PassportVendorLookup map[model.PassportVendorName]model.PassportVendor
	WorkflowWithNodes    model.WorkflowWithNodesCredential
	Cipher               crypto.CryptoCipher
	ServerHost           *serverhost.ServerHost
	MaxSteps             int

	// required by confirm node
	WorkflowInstanceID string
	MailSender         *smtp.Sender

	// testRunNodeID if this node in foreach node, set this foreach node output as Iter data.
	testRunNodeID string
	// if testRunNode in foreach node, testRunNodeID required.
	testForeachNodeID string
}

func NewWorkflowContext(ctx context.Context, opt ContextOpt) (c *WorkflowContext, err error) {
	if opt.MaxSteps == 0 {
		opt.MaxSteps = defaultMaxSteps
	}

	c = &WorkflowContext{
		db:                   opt.DB,
		cache:                opt.Cache,
		cipher:               opt.Cipher,
		serverHost:           opt.ServerHost,
		mailSender:           opt.MailSender,
		useExternalInput:     true,
		flowType:             workflowType,
		logger:               log.Clone(log.Namespace("workflow/context"), log.String("workflow", fmt.Sprintf("<%s:%s>", opt.WorkflowWithNodes.ID, opt.WorkflowWithNodes.Name))),
		context:              ctx,
		workflow:             opt.WorkflowWithNodes,
		executionMode:        executionNormalMode,
		nodeLookup:           map[string]*nodeInstance{},
		scope:                newContextScope(),
		workflowInstanceID:   opt.WorkflowInstanceID,
		subflowContext:       map[string][]*WorkflowContext{},
		maxStepNums:          opt.MaxSteps,
		passportVendorLookup: opt.PassportVendorLookup,
	}

	err = c.buildNodeLookup()
	if err != nil {
		err = fmt.Errorf("building node lookup: %w", err)
		return
	}

	return
}

func NewResumingWorkflowContext(ctx context.Context, opt ContextOpt) (c *WorkflowContext, err error) {
	if opt.MaxSteps == 0 {
		opt.MaxSteps = defaultMaxSteps
	}

	initCtx, cancelInitCtx := context.WithTimeout(ctx, 5*time.Second)
	defer cancelInitCtx()
	instance, err := opt.DB.GetWorkflowInstanceWithNodesByID(initCtx, opt.WorkflowInstanceID)
	if err != nil {
		err = fmt.Errorf("querying workflow instance with nodes by id %q: %w", opt.WorkflowInstanceID, err)
		return
	}

	scope := newContextScope()
	c = &WorkflowContext{
		stepNums:             instance.Steps,
		db:                   opt.DB,
		cache:                opt.Cache,
		cipher:               opt.Cipher,
		serverHost:           opt.ServerHost,
		mailSender:           opt.MailSender,
		useExternalInput:     true,
		flowType:             workflowType,
		logger:               log.Clone(log.Namespace("workflow/context"), log.String("workflow", fmt.Sprintf("<%s:%s>", opt.WorkflowWithNodes.ID, opt.WorkflowWithNodes.Name))),
		context:              ctx,
		workflow:             opt.WorkflowWithNodes,
		executionMode:        executionNormalMode,
		nodeLookup:           map[string]*nodeInstance{},
		scope:                scope,
		workflowInstanceID:   opt.WorkflowInstanceID,
		subflowContext:       map[string][]*WorkflowContext{},
		maxStepNums:          opt.MaxSteps,
		passportVendorLookup: opt.PassportVendorLookup,
	}

	err = c.buildNodeLookup()
	if err != nil {
		err = fmt.Errorf("building node lookup: %w", err)
		return
	}

	for _, node := range instance.Nodes {
		var input, output any

		err = json.Unmarshal(node.Input, &input)
		if err != nil {
			err = fmt.Errorf("unmarshaling input of node %q of workflow instance %q: %w", node.NodeID, instance.ID, err)
			return
		}
		err = json.Unmarshal(node.Output, &output)
		if err != nil {
			err = fmt.Errorf("unmarshaling output of node %q of workflow instance %q: %w", node.NodeID, instance.ID, err)
			return
		}

		// restore scope input/output
		scope.setNodeData(node.NodeID, map[string]any{
			"input":  input,
			"output": output,
		})

		// restore node instance state
		nodeInstance, ok := c.nodeLookup[node.NodeID]
		if !ok {
			// node not existed anymore
			continue
		}

		nodeInstance.input = input
		nodeInstance.output = output
		nodeInstance.executed = true
		nodeInstance.success = node.Status == model.WorkflowInstanceNodeStatusCompleted
		nodeInstance.durationMs = node.DurationMs
	}

	return
}

// ExecutionResult is reported by ExecutionResult.Run
type ExecutionResult struct {
	// required, status of the execution
	Status model.WorkflowInstanceStatus
	// required, how much time does the execution take
	Duration time.Duration
	// required, how many steps does the execution take
	Steps int

	// the id of failing node
	// required if the status is not WorkflowInstanceStatusCompleted
	FailNodeID string
	// the error causes failing
	// required if the status is not WorkflowInstanceStatusCompleted
	Err error

	// each node has a workflow_instance_node record,
	// but InstanceNodes just contains the executed nodes.
	InstanceNodes []*model.WorkflowInstanceNode
}

func (r ExecutionResult) Error() string {
	if r.Err == nil {
		return ""
	}

	return r.Err.Error()
}

// Run starts the execution of the workflow, which starts at startNodeName,
// with input as its input.
//
// Depending on the start node type, a valid input can be:
// * For webhook trigger node, a trigger.HTTPRequest should be used.
//
// Ref: WorkflowContext.bindInitialInput()
// Please, amend here if there's more types. It takes me a lot of time to figure out the info above.
func (w *WorkflowContext) Run(startNodeName string, input any) (result ExecutionResult) {
	var (
		err       error
		startedAt = time.Now()
	)
	defer func() {
		if err == nil {
			w.status = model.WorkflowInstanceStatusCompleted
		} else {
			w.status = model.WorkflowInstanceStatusFailed
		}

		result = ExecutionResult{
			Status:   w.status,
			Duration: time.Since(startedAt),
			Steps:    w.stepNums,
			Err:      err,
		}
		if err != nil {
			result.FailNodeID = w.currentNode.ID
		}

		for _, nodeIns := range w.nodeLookup {
			var nodeInputBytes, nodeOutputBytes []byte
			if !nodeIns.executed {
				continue
			}
			nodeInputBytes, _ = json.Marshal(nodeIns.input)
			nodeOutputBytes, _ = json.Marshal(nodeIns.output)
			nodeMeta := nodeIns.Node.UltrafoxNode()
			status := model.WorkflowInstanceNodeStatusCompleted
			if !nodeIns.success {
				status = model.WorkflowInstanceNodeStatusFailed
			}
			var randomSampleVersion string
			randomSampleVersion, err = utils.ShortNanoID()
			if err != nil {
				err = fmt.Errorf("random sample version: %w", err)
				return
			}
			result.InstanceNodes = append(result.InstanceNodes, &model.WorkflowInstanceNode{
				WorkflowID:         w.workflow.ID,
				WorkflowInstanceID: w.workflowInstanceID,
				NodeID:             nodeIns.ID,
				Status:             status,
				Class:              nodeMeta.Class,
				DurationMs:         nodeIns.durationMs,
				Input:              nodeInputBytes,
				Output:             nodeOutputBytes,
				StartTime:          nodeIns.startTime,
				Source:             model.NodeSourceLive,
				SampleResourceID:   nodeIns.ID,
				SampleVersion:      randomSampleVersion,
			})
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

	w.startNodeName = startNodeName
	if w.stepNums > 0 {
		w.nextNodeName = startNodeName
	}

	w.logger.For(w.context).Debug("workflow start execute")
	opt := oteltrace.WithAttributes(
		attribute.String("workflow", w.workflow.Name),
		attribute.String("startNode", startNodeName))
	ctx, span := otel.Tracer("ultrafox").Start(w.context, "workflow.run", opt)
	w.context = ctx
	defer span.End()

	if len(w.workflow.Nodes) == 0 {
		w.logger.For(w.context).Debug("no nodes in workflow")
		return
	}

	w.initialInput = input
	w.status = model.WorkflowInstanceStatusRunning

	err = w.run()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return
}

// UseInputFields set use own inputFields
func (w *WorkflowContext) UseInputFields() {
	w.useExternalInput = false
}

func (w *WorkflowContext) UseExternalInput() {
	w.useExternalInput = true
}

func (w *WorkflowContext) run() error {
	for w.next() {
		if err := w.contextAborted(); err != nil {
			return err
		}

		w.stepNums++
		if w.stepNums > w.maxStepNums {
			return fmt.Errorf("max step number %d exceeded", w.maxStepNums)
		}
		if err := w.runCurrent(); err != nil {
			return fmt.Errorf("running node: %w", err)
		}

		if w.isTestMode() {
			break
		}

		w.logger.For(w.context).Debug("node executed", log.String("node", w.currentNode.ID))
		w.advance()
	}

	return nil
}

// runSubflow provider for foreach run a item
// cannot change the scope contexts
func (w *WorkflowContext) runSubflow(startNode string) (map[string]any, error) {
	subflowContext := w.newSubflowContext()
	subflowContext.startNodeName = startNode
	subflowContext.status = model.WorkflowInstanceStatusRunning
	err := subflowContext.run()
	if err != nil {
		w.status = model.WorkflowInstanceStatusFailed
		return nil, err
	}

	w.status = model.WorkflowInstanceStatusCompleted
	w.subflowContext[startNode] = append(w.subflowContext[startNode], subflowContext)
	w.stepNums += subflowContext.stepNums
	result := subflowContext.scope.getDiffNodeData()
	return result, nil
}

func (w *WorkflowContext) newSubflowContext() *WorkflowContext {
	workflowCtx := &WorkflowContext{
		db:                   w.db,
		cache:                w.cache,
		flowType:             subflowType,
		logger:               w.logger.With(log.Bool("subflow", true)),
		context:              w.context,
		workflow:             w.workflow,
		nodeLookup:           w.nodeLookup,
		workflowInstanceID:   w.workflowInstanceID,
		scope:                w.scope.cloneProxy(),
		executionMode:        w.executionMode,
		serverHost:           w.serverHost,
		maxStepNums:          w.maxStepNums - w.stepNums,
		mailSender:           w.mailSender,
		passportVendorLookup: w.passportVendorLookup,
	}
	return workflowCtx
}

func (w *WorkflowContext) contextAborted() error {
	done := w.context.Done()
	select {
	case <-done:
		return fmt.Errorf("abort when running node: %w", w.context.Err())
	default:
		return nil
	}
}

func (w *WorkflowContext) runNodeInstance(nodeIns *nodeInstance) (output any, err error) {
	meta := nodeIns.UltrafoxNode()
	nodeContext := w.newNodeContext(nodeIns)
	provisioner, ok := nodeIns.Node.(Provisioner)
	if ok {
		deps := ProvisionDeps{
			Authorizer: nodeIns.authorizer,
			KVProcessor: KVProcessor{
				workflowID: w.workflow.ID,
				db:         w.db,
			},
		}
		if strings.HasPrefix(nodeIns.Class, "ultrafox/") {
			deps.PassportVendorLookup = w.passportVendorLookup
		}

		err = provisioner.Provision(w.context, deps)
		if err != nil {
			err = fmt.Errorf("provisioning %q: %w", meta.Class, err)
			return
		}
	}

	output, err = nodeIns.Run(nodeContext)
	if err != nil {
		err = fmt.Errorf("running on node %q(class: %s): %w", nodeIns.ID, meta.Class, err)
		return
	}
	return
}

func (w *WorkflowContext) queryFieldResultList(nodeIns *nodeInstance) (result QueryFieldResult, err error) {
	meta := nodeIns.UltrafoxNode()
	nodeContext := w.newNodeContext(nodeIns)
	provisioner, ok := nodeIns.Node.(Provisioner)
	if ok {
		deps := ProvisionDeps{
			Authorizer: nodeIns.authorizer,
			KVProcessor: KVProcessor{
				workflowID: w.workflow.ID,
				db:         w.db,
			},
		}
		if strings.HasPrefix(nodeIns.Class, "ultrafox/") {
			deps.PassportVendorLookup = w.passportVendorLookup
		}

		err = provisioner.Provision(w.context, deps)
		if err != nil {
			err = fmt.Errorf("provisioning %q: %w", meta.Class, err)
			return
		}
	}

	provider, ok := nodeIns.Node.(QueryFieldResultProvider)
	if !ok {
		err = fmt.Errorf("class %q don't implement QueryFieldResultProvider", meta.Class)
		return
	}

	result, err = provider.QueryFieldResultList(nodeContext)
	if err != nil {
		err = fmt.Errorf("running on node %q(class: %s): %w", nodeIns.ID, meta.Class, err)
		return
	}

	// if current page no results, marks noMore for stopping load.
	if len(result.Items) == 0 {
		result.NoMore = true
	}
	return
}

func (w *WorkflowContext) runCurrent() (err error) {
	meta := w.currentNode.UltrafoxNode()
	opt := oteltrace.WithAttributes(attribute.String("nodeID", w.currentNode.ID))
	_, createSpan := otel.Tracer("ultrafox").Start(w.context, fmt.Sprintf("node.run: %s", meta.Class), opt)
	defer createSpan.End()
	startTime := time.Now()

	var output, input any

	defer func() {
		duration := utils.NowHumanDurationFrom(startTime)

		w.currentNode.startTime = startTime
		w.currentNode.input = w.currentNode.Node // the original input in Node
		w.currentNode.executed = true
		w.currentNode.durationMs = duration.Milliseconds()

		if submitErr := w.submitNodeData(input, output); submitErr != nil {
			err = fmt.Errorf("submit node output invalid: %w", submitErr)
		}
		if err == nil {
			w.currentNode.success = true
		}
		w.currentNode.output = output
	}()

	// if custom node implement the MetaUnmarshaler, use this for unmarshal, read scope data by node self.
	// now the switch node use this.
	if adapter.IsAnySchema(w.currentNode.inputSchema) {
		err = w.BindInputFieldsToNode(w.currentNode)
		if err != nil {
			return fmt.Errorf("unmarshal node from raw parameters: %w", err)
		}
	} else {
		if w.useExternalInput && w.stepNums == 1 && w.flowType == workflowType {
			// in the first step, input is constant
			err = w.bindInitialInput()
			if err != nil {
				return fmt.Errorf("binding initial input: %w", err)
			}
		} else {
			err = w.unmarshalNodeFromContext(w.currentNode)
			if err != nil {
				return fmt.Errorf("build node %q(class: %s) input: %w", w.currentNode.ID, meta.Class, err)
			}
		}
	}

	input, err = w.submitInput(w.currentNode.Node)
	if err != nil {
		return fmt.Errorf("submit node input invalid: %w", err)
	}

	output, err = w.runNodeInstance(w.currentNode)
	if err != nil {
		createSpan.SetStatus(codes.Error, err.Error())
		return
	}
	createSpan.SetStatus(codes.Ok, "")

	return
}

func (w *WorkflowContext) bindInitialInput() error {
	if w.initialInput == nil {
		return nil
	}
	if b, ok := w.initialInput.([]byte); ok {
		err := json.Unmarshal(b, w.currentNode.Node)
		if err != nil {
			return fmt.Errorf("node %q data invalid: %w", w.currentNode.ID, err)
		}
		return nil
	}

	v := reflect.ValueOf(w.initialInput)
	switch v.Kind() {
	case reflect.String:
		b := []byte(v.String())
		err := json.Unmarshal(b, w.currentNode.Node)
		if err != nil {
			return fmt.Errorf("node %q data invalid: %w", w.currentNode.ID, err)
		}
	default:
		b, err := json.Marshal(w.initialInput)
		if err != nil {
			return fmt.Errorf("node %q data invalid: %w", w.currentNode.ID, err)
		}
		err = json.Unmarshal(b, w.currentNode.Node)
		if err != nil {
			return fmt.Errorf("node %q data invalid: %w", w.currentNode.ID, err)
		}
	}
	return nil
}

func (w *WorkflowContext) submitNodeData(input any, output any) (err error) {
	outputMap, err := trans.TransformToNodeData(output)
	if err != nil {
		err = fmt.Errorf("transform output to map: %w", err)
		return
	}

	w.scope.setNodeData(w.currentNode.ID, map[string]any{
		"input":  input,
		"output": outputMap,
	})
	return
}

func (w *WorkflowContext) submitInput(node Node) (input any, err error) {
	input, err = trans.TransformToNodeData(node)
	if err != nil {
		err = fmt.Errorf("transform node to map: %w", err)
		return
	}

	w.scope.setNodeData(w.currentNode.ID, map[string]any{"input": input})
	return
}

func (w *WorkflowContext) next() bool {
	var (
		nodeIns  *nodeInstance
		ok       bool
		nodeName string
	)

	if w.stepNums == 0 {
		nodeName = w.startNodeName
	} else {
		nodeName = w.nextNodeName
	}

	if nodeName == "" {
		return false
	}
	nodeIns, ok = w.nodeLookup[nodeName]

	if !ok {
		log.For(w.context).Error("node not found", log.String("node", nodeName))
		return false
	}

	w.currentNode = nodeIns
	return true
}

func (w *WorkflowContext) advance() {
	if w.lostControl {
		w.lostControl = false
		return
	}
	w.nextNodeName = w.currentNode.Transition
	w.currentNode = nil
}

func (w *WorkflowContext) buildNodeLookup() error {
	nodes := w.workflow.Nodes

	res := make(map[string]*nodeInstance, len(nodes))
	for _, node := range nodes {
		nodeIns, err := NewNodeInstance(w.context, w.db, w.cache, w.cipher, node)
		if err != nil {
			err = fmt.Errorf("building NodeInstance: %w", err)
			return err
		}
		res[node.ID] = nodeIns
	}
	w.nodeLookup = res

	return nil
}

func (w *WorkflowContext) Context() context.Context {
	return w.context
}

func (w *WorkflowContext) RenderTemplate(content string) ([]byte, error) {
	templateEngine := template.NewTemplateEngine(w.scope)
	return templateEngine.RenderTemplate(content)
}

func (w *WorkflowContext) evaluate(expr string) (result any, err error) {
	// result, err = gval.Evaluate(expr, c.scope)
	return w.scope.evaluate(expr)
}

func (w *WorkflowContext) borrowControl() {
	w.lostControl = true
}

func (w *WorkflowContext) setIterKeyValue(key string, value any) {
	w.scope.setIterKeyValue(key, value)
}

func (w *WorkflowContext) setCurrentIteration(index int, value any, isLast bool) {
	w.scope.setIterKeyValue(iterIndexKey, index)
	w.scope.setIterKeyValue(iterItemKey, value)
	w.scope.setIterKeyValue(iterIsLastKey, isLast)
}

func (w *WorkflowContext) clearScopeIter() {
	w.scope.clearIter()
}

func (w *WorkflowContext) LookupScopeNodeData(nodeName string) any {
	data := w.scope.getNodeData(nodeName)
	if data == nil {
		return nil
	}

	return data.(map[string]any)["output"]
}

func (w *WorkflowContext) BindInputFieldsToNode(nodeIns *nodeInstance) (err error) {
	inputBytes, err := json.Marshal(nodeIns.InputFields)
	if err != nil {
		err = fmt.Errorf("json marshal InputFields: %w", err)
		return
	}

	err = json.Unmarshal(inputBytes, nodeIns.Node)
	if err != nil {
		err = fmt.Errorf("node instance %q data invalid: %w", nodeIns.ID, err)
	}
	return
}

func (w *WorkflowContext) renderInputParameters(node *nodeInstance) ([]byte, error) {
	// need know every input field type
	inputSchema := node.inputSchema
	if inputSchema == nil {
		return nil, nil
	}

	// get input field value by calc with contextScope
	// check input field value type
	reader := newContextScopeReader(w.scope)
	inputBytes, err := schema.RenderJSON(inputSchema, node.InputFields, reader)
	if err != nil {
		return nil, fmt.Errorf("render node input template failed: %w", err)
	}
	return inputBytes, nil
}

// unmarshalNodeFromContext unmarshal node from context.
// calc input fields from context
// render parameter template by cue
func (w *WorkflowContext) unmarshalNodeFromContext(node *nodeInstance) error {
	inputBytes, err := w.renderInputParameters(node)
	if err != nil {
		return err
	}
	if len(inputBytes) == 0 {
		return nil
	}

	err = json.Unmarshal(inputBytes, node.Node)
	if err != nil {
		return fmt.Errorf("decode object failed: %w", err)
	}

	return nil
}

func (w *WorkflowContext) setTestMode() {
	w.executionMode = executionTestMode
}

func (w *WorkflowContext) isTestMode() bool {
	return w.executionMode == executionTestMode
}

// if input cannot match regOnlyVariableReference, matched will be false
func trimBraceBrackets(input string) (reference string, matched bool) {
	subMatches := regOnlyVariableReference.FindStringSubmatch(input)
	if len(subMatches) != 2 {
		return input, false
	}
	return subMatches[1], true
}

func (w *WorkflowContext) getIteration(path string) ([]any, error) {
	v, err := w.evaluate(path)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate path %q: %w", path, err)
	}

	list, err := trans.ToAnySlice(v)
	if err != nil {
		// if path is not a list, build a list by this value.
		return []any{v}, nil
	}
	return list, nil
}
