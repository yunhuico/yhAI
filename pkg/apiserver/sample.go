package apiserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"

	"github.com/getsentry/sentry-go"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/schema"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
)

type nodeMaterial struct {
	node        model.Node
	adapterSpec *adapter.Spec
}

func (h *APIHandler) prepareAllForNode(c *gin.Context, workflowID, nodeID string, permission permission.Permission) (result nodeMaterial, err error) {
	ctx := c.Request.Context()
	workflow, err := h.db.GetWorkflowWithNodesByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow %s: %w", workflowID, err)
		return
	}

	node, ok := workflow.GetNodeByID(nodeID)
	if !ok {
		err = fmt.Errorf("node %s not exists", nodeID)
		return
	}

	currentUserID := getSession(c).UserID
	err = h.enforcer.EnsurePermissions(ctx, currentUserID, workflow.OwnerRef, permission)
	if err != nil {
		_ = c.Error(fmt.Errorf("ensuring permissions: %w", err))
		err = errNoPermissionError
		return
	}

	adapterManager := adapter.GetAdapterManager()
	spec := adapterManager.LookupSpec(node.Class)
	if spec == nil {
		err = fmt.Errorf("unknown adapter spec class %q", node.Class)
		return
	}

	result = nodeMaterial{
		node:        node,
		adapterSpec: spec,
	}
	return
}

// GetAllNodeSamples get all node samples.
// @description each node returns the selected sample.
// @Produce json
// @Param   id  path string true "workflow id"
// @Success 200 {object} apiserver.R{data=response.WorkflowSamplesResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/allNodeSamples [get]
func (h *APIHandler) GetAllNodeSamples(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		workflowID = c.Params.ByName("id")
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	workflow, err := h.db.GetWorkflowWithNodesByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow %s: %w", workflowID, err)
		return
	}
	nodesMap := workflow.Nodes.MapByID()

	currentUserID := getSession(c).UserID
	err = h.enforcer.EnsurePermissions(ctx, currentUserID, workflow.OwnerRef, permission.WorkflowRead)
	if err != nil {
		_ = c.Error(fmt.Errorf("ensuring permissions: %w", err))
		err = errNoPermissionError
		return
	}

	samples, err := h.db.GetSelectedSampleByWorkflowID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("get selected sample by workflow: %w", err)
		return
	}

	resp := response.WorkflowSamplesResp{
		Samples: map[string]response.SampleData{},
	}

	manager := adapter.GetAdapterManager()

	for _, sample := range samples {
		if _, ok := resp.Samples[sample.NodeID]; ok {
			continue
		}

		node, ok := nodesMap[sample.NodeID]
		if !ok {
			h.logger.For(ctx).Warn("sample related node not found", log.String("workflowID", workflowID), log.Int("sampleID", sample.ID))
			continue
		}

		// indicates the node.class changed.
		if node.Class != sample.Class {
			continue
		}

		spec := manager.LookupSpec(node.Class)
		if spec == nil {
			h.logger.For(ctx).Warn("node class not found", log.String("workflowID", workflowID),
				log.Int("sampleID", sample.ID),
				log.String("class", node.Class))
			continue
		}

		data, ok := buildSampleData(ctx, nodesMap, sample.NodeID, sample, 0)
		if !ok {
			continue
		}

		// recover all legacy nodes which testingStatus are empty in this api.
		// update testStatus to success.
		if node.TestingStatus == "" {
			err = processNodeTestingStatusTransition(ctx, h.db.Operator, &node, model.TestNodeSuccessfully)
			if err != nil {
				err = fmt.Errorf("processing node testing status: %w", err)
				return
			}
		}

		if !node.TestSuccessedOrSkipped() {
			continue
		}

		resp.Samples[sample.NodeID] = data
	}

	OK(c, resp)
}

// GetNodeSamples get node samples.
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Success 200 {object} apiserver.R{data=response.NodeSamplesResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/samples [get]
func (h *APIHandler) GetNodeSamples(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		workflowID = c.Params.ByName("id")
		nodeID     = c.Params.ByName("nodeId")
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	workflow, err := h.db.GetWorkflowWithNodesByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow %s: %w", workflowID, err)
		return
	}
	nodesMap := workflow.Nodes.MapByID()

	preparedNodeMaterial, err := h.prepareAllForNode(c, workflowID, nodeID, permission.WorkflowRead)
	if err != nil {
		return
	}
	node := preparedNodeMaterial.node
	spec := preparedNodeMaterial.adapterSpec

	var (
		samples model.WorkflowInstanceNodes
		resp    response.NodeSamplesResp
	)

	defer func() {
		if err != nil {
			return
		}
		OK(c, resp)
	}()

	if !node.TestSuccessedOrSkipped() {
		return
	}

	// get current selected samples
	var selectedSample model.WorkflowInstanceNode
	selectedSample, err = h.db.GetSelectedSampleByWorkflowIDAndNode(ctx, workflowID, node.ID, node.Class)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return
	}

	// returns up to 10 data samples, merge the selectedSample into samples if samples don't contains it.
	mergeSamples := func() {
		if selectedSample.ID == 0 {
			return
		}

		if len(samples) == 0 {
			samples = model.WorkflowInstanceNodes{&selectedSample}
			return
		}

		for _, sample := range samples {
			// if samples contains selected sample, do nothing.
			if sample.ID == selectedSample.ID {
				return
			}
		}

		if len(samples) == 10 {
			samples[9] = &selectedSample
		} else {
			samples = append(samples, &selectedSample)
		}
	}

	// if not exists a selected sample, choose the latest sample as selected.
	selectLatestSampleIfNoSelectedSample := func() (err error) {
		if selectedSample.ID == 0 && len(samples) > 0 {
			// update the latest sample to selected.
			err = h.db.UpdateSampleToSelectedByID(ctx, samples[0].ID)
			if err != nil {
				err = fmt.Errorf("update sample workflow instance to selected: %w", err)
				return
			}
			samples[0].IsSelectedSample = true
		}
		return
	}

	if !spec.IsTrigger() {
		// other nodes just returns current sample.
		samples = model.WorkflowInstanceNodes{&selectedSample}
	} else {
		samples, err = h.db.GetLatestSampleByNodeAndLimit(ctx, nodeID, node.Class, 10)
		if err != nil {
			err = fmt.Errorf("get latest sample workflow instance nodes: %w", err)
			return
		}

		if err = selectLatestSampleIfNoSelectedSample(); err != nil {
			return
		}

		mergeSamples()
	}

	for i, sample := range samples {
		data, ok := buildSampleData(ctx, nodesMap, node.ID, sample, i)
		if !ok {
			continue
		}
		resp.Samples = append(resp.Samples, data)
	}
}

func buildSampleData(
	ctx context.Context,
	nodesMap map[string]model.Node,
	nodeID string,
	sample *model.WorkflowInstanceNode,
	sampleIndex int,
) (data response.SampleData, ok bool) {
	ok = true
	node := nodesMap[nodeID]
	adapterManager := adapter.GetAdapterManager()
	spec := adapterManager.LookupSpec(node.Class) // the spec must exists.
	sampleNamePrefix := "Sample"
	if spec.ShortName != "" {
		sampleNamePrefix = spec.ShortName
	}
	data = response.SampleData{
		SampleID:   sample.ID,
		SampleName: fmt.Sprintf("%s - %d", sampleNamePrefix, sampleIndex+1),
		IsSelected: sample.IsSelectedSample,
		SampledAt:  sample.SampledAt,
		Source:     sample.Source,
	}

	var (
		rawOutput     any
		flattenOutput []schema.OutputField // nolint: staticcheck
		ignoreErr1    error
		ignoreErr2    error
	)

	defer func() {
		for _, err := range []error{ignoreErr1, ignoreErr2} {
			if err != nil {
				ok = false
				event := &sentry.Event{
					Level:   sentry.LevelInfo,
					Message: err.Error(),
				}
				sentry.CaptureEvent(event)
			}
		}
	}()

	// if this sample created by skipped test, no output
	rawOutput, ignoreErr1 = trans.TransformToNodeData(sample.Output)
	data.RunNodeResp.RawOutput = rawOutput

	if sample.Source == model.NodeSourceSkip {
		data.SampleName = fmt.Sprintf("Skipped - %d", sampleIndex+1)
		return
	}

	flattenOutput, ignoreErr2 = buildFlattenOutput(ctx, nodesMap, node.ID, rawOutput)
	data.RunNodeResp.FlattenOutput = flattenOutput
	return
}

// DEPRECATED
func buildFlattenOutput(
	ctx context.Context,
	nodeMap map[string]model.Node,
	nodeID string,
	output any,
) (flattenOutput []schema.OutputField, err error) { // nolint: staticcheck
	adapterManager := adapter.GetAdapterManager()
	node := nodeMap[nodeID] // node must exists.
	spec := adapterManager.LookupSpec(node.Class)

	runNodeIsForeachNode := node.Class == validate.ForeachClass
	if !runNodeIsForeachNode {
		// use `.Node.{nodeID}.output` as prefix.
		flattenOutput = schema.BuildOutput(ctx, spec.OutputSchema, node.ID, output) // nolint: staticcheck
		return
	}

	// foreach test output, reference prefix is `.Iter.`
	// if foreach-inside-node use foreach variable, examples like `.Iter.loopIteration`, `.Iter.loopItems`,
	// use `.Iter` as prefix.
	flattenOutput = schema.BuildForeachOutput(ctx, spec.OutputSchema, output) // nolint: staticcheck
	// second-part flatten output from `list data`, examples like `Issue list`, `Merge request list`,
	// should merge the loopItem's flatten fields to flattenOutput.
	// so parse the inputCollection expression, get the node which is the list-data from and the key of list field.
	// expression like `{{ .Node.{nodeID}.output }}`, `{{ .Node.{nodeID}.output.list }}`
	// or `.Node.{nodeID}.output.list`
	expr := node.Data.InputFields["inputCollection"].(string)
	referenceNodeID, referenceKeyPath, ok := workflow.ParseNodeOutputVariableReferenceExpression(expr) // nolint: staticcheck
	if !ok {
		return
	}
	referenceNode, exists := nodeMap[referenceNodeID]
	if !exists {
		err = fmt.Errorf("input collection reference node %s does not exist", referenceNodeID)
		return
	}
	var referenceNodeFlattenOutput []schema.OutputField // nolint: staticcheck
	referenceNodeSpec := adapterManager.LookupSpec(referenceNode.Class)
	for _, schemaField := range referenceNodeSpec.OutputSchema {
		if schemaField.Key != referenceKeyPath {
			continue
		}

		// foreach-field must define the child's fields.
		if schemaField.Child == nil {
			return
		}

		var loopItem any
		// foreach output is map.
		outputMap, isMap := output.(map[string]any)
		if isMap {
			loopItem = outputMap["loopItem"]
		}

		// indicate use choose this field as foreach's list data.
		// get the list data from node's sample data.
		referenceNodeFlattenOutput = schema.BuildForeachOutput( // nolint: staticcheck
			ctx,
			schemaField.Child.Fields,
			map[string]any{
				"loopItem": loopItem,
			} /* iter item data */)

		flattenOutput = append(flattenOutput, referenceNodeFlattenOutput...)
		break
	}

	return
}

// LoadMoreSamples load more node samples.
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/samples/loadMore [post]
func (h *APIHandler) LoadMoreSamples(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		workflowID = c.Params.ByName("id")
		nodeID     = c.Params.ByName("nodeId")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		OK(c, nil)
	}()

	preparedNodeMaterial, err := h.prepareAllForNode(c, workflowID, nodeID, permission.WorkflowRun)
	if err != nil {
		return
	}
	node := preparedNodeMaterial.node

	// only trigger node can reload samples, but we ignore this exception.
	if node.Type != model.NodeTypeTrigger {
		OK(c, nil)
		return
	}

	loader := h.newSampleLoader(ctx, workflowID, node, preparedNodeMaterial)
	defer func() {
		// The processing testingStatus has the highest priority.
		currentSamples, _ := h.db.GetLatestSampleByNodeAndLimit(ctx, node.ID, node.Class, 1)
		// if the trigger node exists valid samples, update the testingStatus to success.
		if len(currentSamples) == 1 {
			_ = processNodeTestingStatusTransition(ctx, h.db.Operator, &node, model.TestNodeSuccessfully)
		}

		if loader.err != nil {
			err = loader.err
			return
		}

		// if no new samples, the frontend need to know the information so that indicate user.
		if !loader.hasNewSamples {
			err = errNoMoreSamplesToLoad
			return
		}
	}()

	loader.convertFromLiveInstanceNodes()
	loader.loadComposeSamples()
	loader.insertLatestComposeSamples()
}

type sampleLoader struct {
	ctx                  context.Context
	logger               log.Logger
	workflowID           string
	node                 model.Node
	db                   *model.DB
	cache                *cache.Cache
	preparedNodeMaterial nodeMaterial
	sampleProviders      map[string]trigger.SampleProvider
	cipher               crypto.CryptoCipher
	passportVendorLookup map[model.PassportVendorName]model.PassportVendor

	samples       trigger.SampleList
	err           error
	hasNewSamples bool
	over          bool
}

func (h *APIHandler) newSampleLoader(ctx context.Context, workflowID string, node model.Node, material nodeMaterial) *sampleLoader {
	return &sampleLoader{
		ctx:                  ctx,
		logger:               h.logger,
		workflowID:           workflowID,
		node:                 node,
		db:                   h.db,
		cache:                h.cache,
		preparedNodeMaterial: material,
		sampleProviders:      workflow.SampleProviders(),
		cipher:               h.cipher,
		passportVendorLookup: h.passportVendorLookup,
		over:                 false,
	}
}

func (l *sampleLoader) convertFromLiveInstanceNodes() {
	// get latest 10 pieces of live instance_nodes
	latestInstanceNodes, err := l.db.GetLatestLiveWorkflowInstanceNodesByWorkflowIDAndNodeID(l.ctx, l.workflowID, l.node.ID, l.node.Class, 10)
	if err != nil {
		l.err = fmt.Errorf("get latest 10 pieces of instance_nodes: %w", err)
		return
	}

	var updateToSampleIDList []int
	for _, instanceNode := range latestInstanceNodes {
		if instanceNode.IsSample {
			continue
		}

		updateToSampleIDList = append(updateToSampleIDList, instanceNode.ID)
	}

	if len(updateToSampleIDList) == 0 {
		return
	}

	err = l.db.UpdateWorkflowInstanceNodesToSample(l.ctx, updateToSampleIDList)
	if err != nil {
		l.err = fmt.Errorf("update workflow instance nodes to sample: %w", err)
		return
	}

	l.hasNewSamples = true
	// if converted 10 pieces of samples, don't call SampleProvider to compose samples.
	if len(updateToSampleIDList) == 10 {
		l.over = true
	}
}

func (l *sampleLoader) shouldReturn() bool {
	if l.over {
		return true
	}
	if l.err != nil {
		return true
	}
	return false
}

func (l *sampleLoader) loadComposeSamples() {
	if l.shouldReturn() {
		return
	}

	// if the trigger node implements SampleProvider, compose the samples by SampleProvider.
	provider, ok := l.sampleProviders[l.node.Class]
	if !ok {
		return
	}

	var authorizer auth.Authorizer
	authorizer, err := l.newAuthorizerForNode(l.ctx, l.node, l.preparedNodeMaterial.adapterSpec)
	if err != nil {
		l.err = fmt.Errorf("new authorizer for trigger node: %w", err)
		return
	}
	configObject := provider.GetConfigObject()
	if configObject != nil {
		var decoder *mapstructure.Decoder
		decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Squash: true,
			Result: configObject,
		})
		if err != nil {
			l.err = fmt.Errorf("initializing decoder: %w", err)
			return
		}

		err = decoder.Decode(l.node.Data.InputFields)
		if err != nil {
			l.err = fmt.Errorf("bind node input fields to trigger config object: %w", err)
			return
		}
	}

	var passportVendorLookup map[model.PassportVendorName]model.PassportVendor
	if strings.HasPrefix(l.node.Class, "ultrafox/") {
		passportVendorLookup = l.passportVendorLookup
	}

	var samples trigger.SampleList
	samples, err = provider.GetSampleList(trigger.NewBaseContext(l.ctx, authorizer, configObject, passportVendorLookup))
	if err != nil {
		l.err = fmt.Errorf("get sample list: %w", err)
		return
	}
	l.samples = samples

	if len(samples) == 0 {
		l.over = true
		return
	}
}

func (l *sampleLoader) insertLatestComposeSamples() {
	if l.shouldReturn() {
		return
	}

	if len(l.samples) == 0 {
		return
	}

	var sampleWorkflowInstanceNodes = make(model.WorkflowInstanceNodes, 0, len(l.samples))
	var composeSamplesExistsMap map[string]bool
	composeSamplesExistsMap, err := l.db.SelectComposeSampleExistsByResourceIDAndVersion(l.ctx, l.node.ID, l.samples.GetResourcePairs())
	if err != nil {
		l.err = fmt.Errorf("select sample exists by resource id and version: %w", err)
		return
	}

	for _, item := range l.samples {
		if composeSamplesExistsMap[item.GetID()] {
			continue
		}

		var outputBytes []byte
		outputBytes, err = json.Marshal(item)
		if err != nil {
			l.logger.For(l.ctx).Error("marshal sample error", log.ErrField(err))
			continue
		}

		sampleWorkflowInstanceNodes = append(sampleWorkflowInstanceNodes, &model.WorkflowInstanceNode{
			WorkflowID:         l.workflowID,
			WorkflowInstanceID: "", // keep empty, is ok, compose sample has no WorkflowInstanceID
			NodeID:             l.node.ID,
			Status:             model.WorkflowInstanceNodeStatusCompleted,
			Class:              l.node.Class,
			Input:              outputBytes,
			Output:             outputBytes,
			Source:             model.NodeSourceCompose,
			IsSelectedSample:   false,
			IsSample:           true,
			SampleResourceID:   item.GetID(),
			SampledAt:          time.Now(),
			SampleVersion:      item.GetVersion(),
		})
	}

	if len(sampleWorkflowInstanceNodes) == 0 {
		return
	}

	l.hasNewSamples = true
	err = l.db.BulkInsertWorkflowInstanceNodes(l.ctx, sampleWorkflowInstanceNodes)
	if err != nil {
		l.err = err
	}
}

func (l *sampleLoader) newAuthorizerForNode(ctx context.Context, node model.Node, spec *adapter.Spec) (authorizer auth.Authorizer, err error) {
	if !spec.AdapterMeta.RequireAuth() {
		return nil, nil
	}

	if node.CredentialID == "" {
		return nil, fmt.Errorf("node credential ID is required")
	}

	credential, err := l.db.GetCredentialByID(ctx, node.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	authorizer, err = auth.NewAuthorizer(l.cipher, &credential,
		auth.WithUpdateCredentialTokenFunc(l.db.UpdateCredentialTokenAndConfirmStatusByID),
		auth.WithOAuthCredentialUpdater(auth.OAuthCredentialUpdater{DB: l.db.Operator, Cache: l.cache}),
	)
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	return
}

// SelectNodeSample choose the given sample as selected.
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Param   sampleId  path string true "sample id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/samples/{sampleId}/select [post]
func (h *APIHandler) SelectNodeSample(c *gin.Context) {
	var (
		err         error
		ctx         = c.Request.Context()
		workflowID  = c.Params.ByName("id")
		nodeID      = c.Params.ByName("nodeId")
		sampleIDStr = c.Params.ByName("sampleId")
		sampleID    int
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	sampleID, err = strconv.Atoi(sampleIDStr)
	if err != nil {
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	_, err = h.prepareAllForNode(c, workflowID, nodeID, permission.WorkflowRun)
	if err != nil {
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		var chooseSample model.WorkflowInstanceNode
		chooseSample, err = h.db.GetNodeSampleByID(ctx, sampleID)
		if err != nil {
			err = fmt.Errorf("get sample worker workflow instance node: %w", err)
			return
		}

		err = h.db.UpdateSampleToUnselectedByWorkflowIDAndNodeID(ctx, workflowID, nodeID)
		if err != nil {
			err = fmt.Errorf("update all sample worker instance node to unselected: %w", err)
			return
		}

		err = h.db.UpdateSampleToSelectedByID(ctx, chooseSample.ID)
		if err != nil {
			err = fmt.Errorf("update sample worker workflow instance node to selected: %w", err)
			return
		}

		return
	})

	OK(c, nil)
}

// SkipTestNode skip test node.
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/skipTest [post]
func (h *APIHandler) SkipTestNode(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		workflowID = c.Params.ByName("id")
		nodeID     = c.Params.ByName("nodeId")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	preparedNodeMaterial, err := h.prepareAllForNode(c, workflowID, nodeID, permission.WorkflowRun)
	if err != nil {
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		var node model.Node
		node, err = tx.GetNodeByID(ctx, nodeID)
		if err != nil {
			err = fmt.Errorf("getting node: %w", err)
			return
		}

		err = tx.UpdateSampleToUnselectedByWorkflowIDAndNodeID(ctx, workflowID, nodeID)
		if err != nil {
			err = fmt.Errorf("update all sample worker instance node to unselected: %w", err)
			return
		}
		randomSampleVersion, err := utils.ShortNanoID()
		if err != nil {
			err = fmt.Errorf("random sample version: %w", err)
			return
		}
		err = tx.InsertWorkflowInstanceNode(ctx, &model.WorkflowInstanceNode{
			WorkflowID:       workflowID,
			NodeID:           nodeID,
			Status:           model.WorkflowInstanceNodeStatusCompleted,
			Class:            preparedNodeMaterial.node.Class,
			DurationMs:       0,
			Source:           model.NodeSourceSkip,
			Output:           []byte(`{"skipped": true}`),
			IsSelectedSample: true,
			IsSample:         true,
			SampleResourceID: nodeID,
			SampledAt:        time.Now(),
			SampleVersion:    randomSampleVersion,
		})
		if err != nil {
			err = fmt.Errorf("insert workflow instance node: %w", err)
			return
		}

		err = processNodeTestingStatusTransition(ctx, tx, &node, model.UserChooseSkip)
		if err != nil {
			err = fmt.Errorf("processing node testing status transition: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("skip node test: %w", err)
		return
	}

	OK(c, nil)
}
