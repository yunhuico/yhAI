package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

// CreateNode create node
// @Summary create node
// @Produce json
// @Success 200 {object} apiserver.R{data=response.ResourceCreatedResponse}
// @Failure 400 {object} apiserver.R
// @Param   id  path string true "workflow id"
// @Param   body body payload.EditNodeReq true "the payload"
// @Router /api/v1/workflows/{id}/nodes [post]
func (h *APIHandler) CreateNode(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		newNode    *model.Node
		req        payload.EditNodeReq
		workflowID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if err = c.ShouldBindJSON(&req); err != nil {
		err = errBizInvalidRequestPayload
		return
	}

	newNode, err = h.assembleNode(c, req)
	if err != nil {
		err = fmt.Errorf("assembling newNode: %w", err)
		return
	}
	newNode.TestingStatus = model.NodeTestingDefaultStatus

	userWorkflow, err := h.db.GetWorkflowByID(ctx, newNode.WorkflowID)
	if err != nil {
		err = errBizInvalidWorkflowID
		return
	}
	if req.IsStart && userWorkflow.StartNodeID != "" {
		err = fmt.Errorf("start newNode already exists: %w", errCreateNodeDenied)
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	if req.IsStart {
		err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
			err = tx.InsertNode(ctx, newNode)
			if err != nil {
				err = fmt.Errorf("inserting node: %w", err)
				return
			}
			err = tx.UpdateWorkflowStartNodeID(ctx, newNode.WorkflowID, newNode.ID)
			if err != nil {
				err = fmt.Errorf("updating workflow start newNode id: %w", err)
				return
			}

			err = workflow.InitTrigger(ctx, h.triggerRegistry, tx, workflowID, newNode)
			if err != nil {
				err = fmt.Errorf("initing trigger: %w", err)
				return
			}
			return
		})

		if err != nil {
			err = fmt.Errorf("create node in tx: %w", err)
			return
		}

		OK(c, response.ResourceCreatedResponse{
			ID: newNode.ID,
		})
		return
	}

	// create non-starting node
	process, err := newCreateNodeProcess(ctx, newNode, req.PreviousNodeInfo, h.db)
	if err != nil {
		err = fmt.Errorf("initializing node creation process")
		return
	}
	err = process.execute()
	if err != nil {
		err = fmt.Errorf("create node process error: %w", err)
		return
	}

	OK(c, response.ResourceCreatedResponse{
		ID: newNode.ID,
	})
}

// UpdateNode update node
// @Summary update node
// @Produce json
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Param   body body payload.EditNodeReq true "the payload"
// @Router /api/v1/workflows/{id}/nodes/{nodeId} [put]
func (h *APIHandler) UpdateNode(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
		nodeID     = c.Param("nodeId")
		node       *model.Node
		req        payload.EditNodeReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	oldNode, err := h.db.GetNodeByID(ctx, nodeID)
	if err != nil {
		err = fmt.Errorf("querying node by id %q: %w", nodeID, err)
		_ = c.Error(err)
		err = errBizReadDatabase
		return
	}

	if oldNode.WorkflowID != workflowID {
		err = fmt.Errorf("workflow id does not match, want %q, got %q", oldNode.WorkflowID, workflowID)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow by id %q", workflowID)
		_ = c.Error(err)
		err = errBizInvalidWorkflowID
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	if req.IsStart && userWorkflow.Status == model.WorkflowStatusEnabled {
		_ = c.Error(errors.New("workflow is enabled, cannot update trigger node"))
		err = errUpdateNodeDenied
		return
	}

	node, err = h.assembleNode(c, req)
	if err != nil {
		err = fmt.Errorf("assembling node: %w", err)
		return
	}
	node.ID = oldNode.ID
	node.TestingStatus = oldNode.TestingStatus

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.UpdateNodeByID(ctx, node)
		if err != nil {
			err = fmt.Errorf("updating node: %w", err)
			return
		}

		if node.Class != oldNode.Class || !reflect.DeepEqual(node.Data.InputFields, oldNode.Data.InputFields) {
			// update node testStatus
			err = processNodeTestingStatusTransition(ctx, tx, node, model.NodeUpdated)
			if err != nil {
				err = fmt.Errorf("processing node test status transition: %w", err)
				return
			}
		}

		if !req.IsStart {
			// end of execution if updating the actor node.
			return
		}

		// delete the old trigger if trigger-node class changed.
		if oldNode.Class != node.Class {
			err = tx.DeleteTriggersByWorkflowID(ctx, node.WorkflowID)
			if err != nil {
				err = fmt.Errorf("deleting old trigger: %w", err)
				return
			}
		}

		// update workflow.startNodeID to the updating node.
		err = tx.UpdateWorkflowStartNodeID(ctx, node.WorkflowID, node.ID)
		if err != nil {
			err = fmt.Errorf("updating workflow.startNodeID failed: %w", err)
			return
		}

		err = workflow.InitTrigger(ctx, h.triggerRegistry, tx, workflowID, node)
		if err != nil {
			err = fmt.Errorf("initing trigger: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("update node: %w", err)
		return
	}

	OK(c, nil)
}

// UpdateNodeTransition update the node transition
// @Summary update the node transition
// @Produce json
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Param   body body payload.UpdateNodeTransitionReq true "the payload"
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/transition [put]
func (h *APIHandler) UpdateNodeTransition(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
		nodeID     = c.Param("nodeId")
		req        payload.UpdateNodeTransitionReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if err = c.ShouldBindJSON(&req); err != nil {
		err = errBizInvalidRequestPayload
		return
	}

	oldNode, err := h.db.GetNodeByID(ctx, nodeID)
	if err != nil {
		err = errBizReadDatabase
		return
	}

	if oldNode.WorkflowID != workflowID {
		err = errBizInvalidRequestPayload
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = errBizInvalidWorkflowID
		return
	}
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	err = h.db.UpdateNodeTransition(ctx, nodeID, req.Transition)
	if err != nil {
		return
	}

	OK(c, nil)
}

// UpdateSwitchNodePathName update the switch-node path name
// @Produce json
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Param   body body payload.UpdateSwitchNodePathNameReq true "the payload"
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/pathName [put]
func (h *APIHandler) UpdateSwitchNodePathName(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
		nodeID     = c.Param("nodeId")
		req        payload.UpdateSwitchNodePathNameReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if err = c.ShouldBindJSON(&req); err != nil {
		err = errBizInvalidRequestPayload
		return
	}

	node, err := h.db.GetNodeByID(ctx, nodeID)
	if err != nil {
		err = errBizReadDatabase
		return
	}

	if node.WorkflowID != workflowID {
		err = errBizInvalidRequestPayload
		return
	}

	if node.Class != validate.SwitchClass {
		err = errBizInvalidRequestPayload
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = errBizInvalidWorkflowID
		return
	}
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	var (
		switchNodeStruct validate.SwitchLogicNode
		switchNodeMap    map[string]any
	)
	err = trans.MapToStruct(node.Data.InputFields, &switchNodeStruct)
	if err != nil {
		err = fmt.Errorf("switch node input field error: %w", err)
		return
	}
	if req.Index > len(switchNodeStruct.Paths) {
		err = fmt.Errorf("requesting path index is out of range")
		return
	}
	switchNodeStruct.Paths[req.Index].Name = req.Name
	switchNodeMap, err = trans.StructToMap(switchNodeStruct)
	if err != nil {
		err = fmt.Errorf("switch node struct error: %w", err)
		return
	}
	node.Data.InputFields = switchNodeMap

	err = h.db.UpdateNodeDataByID(ctx, &node)
	if err != nil {
		return
	}

	OK(c, nil)
}

func (h *APIHandler) assembleNode(c *gin.Context, payload payload.EditNodeReq) (node *model.Node, err error) {
	var (
		ctx        = c.Request.Context()
		workflowID = c.Param("id")
		credential model.Credential
	)

	if err = payload.Validate(); err != nil {
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow by id %q: %w", workflowID, err)
		return
	}

	adapterManager := adapter.GetAdapterManager()
	if adapterManager.SpecRequireAuth(payload.Class) {
		credential, err = h.db.GetCredentialByID(ctx, payload.CredentialID)
		if err != nil {
			err = fmt.Errorf("querying credential by id %q: %w", payload.CredentialID, err)
			return
		}

		if userWorkflow.OwnerRef != credential.OwnerRef {
			err = fmt.Errorf("workflow(%s) owner(%s %d) does not match credential owner(%s %d)", userWorkflow.ID, userWorkflow.OwnerType, userWorkflow.OwnerID, credential.OwnerType, credential.OwnerID)
			return
		}

		if !strings.HasPrefix(payload.Class, credential.AdapterClass) {
			err = fmt.Errorf("adapter class(%s) does not match credential's adapter class(%s)", payload.Class, credential.AdapterClass)
			return
		}
	} else {
		if payload.CredentialID != "" {
			err = fmt.Errorf("adapter does not need credential but provides a credential ID, got %q", payload.CredentialID)
			return
		}
	}

	node, err = payload.Normalize()
	if err != nil {
		err = fmt.Errorf("normalizing payload: %w", err)
		return
	}
	node.WorkflowID = workflowID
	return
}

func (h *APIHandler) buildContextDataBySample(ctx context.Context, workflowID string) (data map[string]any, err error) {
	data = map[string]any{}
	var samples model.WorkflowInstanceNodes
	samples, err = h.db.GetSelectedSampleByWorkflowID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("get selected sample by workflowID: %w", err)
		return
	}

	for _, sample := range samples {
		var output any
		badLuck := json.Unmarshal(sample.Output, &output)
		if badLuck != nil {
			h.logger.For(ctx).Warn("sample output is invalid",
				log.String("workflowID", workflowID),
				log.Int("sampleID", sample.ID),
				log.ErrField(err))
			continue
		}

		data[sample.NodeID] = output
	}
	return
}

func processNodeTestingStatusTransition(ctx context.Context, tx model.Operator, node *model.Node, event string) (err error) {
	fsm := model.NewNodeTestingFMS(node)
	needsUpdateNode := fsm.SubmitEvent(ctx, event)
	if needsUpdateNode {
		err = tx.UpdateNodeTestingStatus(ctx, node.ID, node.TestingStatus)
		if err != nil {
			err = fmt.Errorf("updating node testing status: %w", err)
			return
		}
	}
	return
}

// RunNode
// @Summary run node
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Param   body body payload.RunNodeReq true "the payload"
// @Success 200 {object} apiserver.R{data=response.RunNodeResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/run [post]
func (h *APIHandler) RunNode(c *gin.Context) {
	const timeout = 20 * time.Second

	var (
		ctx, cancel = context.WithTimeout(c.Request.Context(), timeout)
		workflowID  = c.Param("id")
		nodeID      = c.Param("nodeId")
		err         error
		req         payload.RunNodeReq
	)
	defer func() {
		cancel()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = errBizInvalidRequestPayload
		return
	}

	// 1. get the workflowWithNodes with nodes.
	workflowWithNodes, err := h.db.GetWorkflowWithNodesCredentialByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow: %w", err)
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, workflowWithNodes.OwnerRef, permission.WorkflowRun)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	// 2. check the node exists
	nodesWithCredentialMap := workflowWithNodes.Nodes.MapByID()
	runNode, existed := nodesWithCredentialMap[nodeID]
	if !existed {
		err = errBizInvalidRequestPayload
		return
	}
	adapterManager := adapter.GetAdapterManager()
	spec := adapterManager.LookupSpec(runNode.Class)
	if spec == nil {
		err = fmt.Errorf("unknown adapter spec class %q", runNode.Class)
		return
	}

	// update node testingStatus
	defer func() {
		var event string
		if err != nil {
			event = model.TestNodeFailed
		} else {
			event = model.TestNodeSuccessfully
		}
		// easy, testingStatus is not a crucial property.
		_ = processNodeTestingStatusTransition(ctx, h.db.Operator, &runNode.Node, event)
	}()

	// 3.1 if the node is trigger, tell frontend wait response.
	// cron trigger is a particular case.
	if runNode.Type == model.NodeTypeTrigger && !runNode.IsCronTrigger() {
		err = errors.New("trigger node can't run directly, choose a sample first")
		return
	}
	// 3.2 build the context data from sample data
	contextData, err := h.buildContextDataBySample(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("build context data failed by samlpe: %v", err)
		return
	}

	// 3.3 run node directly
	opt := workflow.TestWorkflowActionOpt{
		BaseWorkflowActionOpt: workflow.BaseWorkflowActionOpt{
			Ctx:                  ctx,
			WorkflowWithNodes:    workflowWithNodes,
			DB:                   h.db,
			Cipher:               h.cipher,
			ServerHost:           h.serverHost,
			MailSender:           h.mailSender,
			PassportVendorLookup: h.passportVendorLookup,
			Cache:                h.cache,
		},
		NodeID:      nodeID,
		ContextData: contextData,
	}
	if req.ParentNodeID != "" {
		foreachNode, ok := nodesWithCredentialMap[req.ParentNodeID]
		if !ok {
			err = errBizInvalidRequestPayload
			return
		}
		opt.ForeachNode = &foreachNode.Node
		opt.IterIndex = req.IterIndex
	}

	testAction, err := workflow.NewTestWorkflowAction(opt)
	if err != nil {
		err = fmt.Errorf("init WorkflowAction: %w", err)
		return
	}
	err = testAction.Run()
	if err != nil {
		err = fmt.Errorf("%w: %s", errRunNodeFailed, err.Error())
		return
	}
	output := testAction.GetWorkflowContext().LookupScopeNodeData(nodeID)

	var randomSampleVersion string
	randomSampleVersion, err = utils.ShortNanoID()
	if err != nil {
		err = fmt.Errorf("random sample version: %w", err)
		return
	}

	nodeOutputBytes, _ := json.Marshal(output)

	// create latest selected sample for this node
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.UpdateSampleToUnselectedByWorkflowIDAndNodeID(ctx, workflowID, nodeID)
		if err != nil {
			err = fmt.Errorf("update node all samples: %w", err)
			return
		}

		err = tx.InsertWorkflowInstanceNode(ctx, &model.WorkflowInstanceNode{
			WorkflowID:       workflowID,
			NodeID:           nodeID,
			Status:           model.WorkflowInstanceNodeStatusCompleted,
			Class:            runNode.Class,
			Output:           nodeOutputBytes,
			StartTime:        time.Now(),
			Source:           model.NodeSourceTest,
			IsSelectedSample: true,
			IsSample:         true,
			SampleResourceID: nodeID,
			SampledAt:        time.Now(),
			SampleVersion:    randomSampleVersion,
		})
		if err != nil {
			err = fmt.Errorf("insert sample: %w", err)
			return
		}
		return
	})

	if err != nil {
		err = fmt.Errorf("update node sample in tx: %w", err)
		return
	}

	flattenOutput, err := buildFlattenOutput(ctx, nodesWithCredentialMap.ToNodesMap(), nodeID, output) // nolint: staticcheck
	OK(c, response.RunNodeResp{
		RawOutput:     output,
		FlattenOutput: flattenOutput,
	})
}

// DeleteNode
// @Summary run node
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/nodes/{nodeId} [delete]
func (h *APIHandler) DeleteNode(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		workflowID = c.Param("id")
		nodeID     = c.Param("nodeId")
		err        error
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = errBizInvalidWorkflowID
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	isDeletingStartNode := userWorkflow.StartNodeID == nodeID

	// can't delete enabled workflow's first node.
	if isDeletingStartNode && userWorkflow.Status == model.WorkflowStatusEnabled {
		err = fmt.Errorf("can not delete start node when workflow is active: %w", errDeleteNodeDenied)
		return
	}
	currentNode, err := h.db.GetNodeByID(ctx, nodeID)
	if err != nil {
		return
	}

	// if trigger node has next node, can't delete it.
	if isDeletingStartNode && currentNode.Transition != "" {
		err = fmt.Errorf("can not delete start node when it has transition node: %w", errDeleteNodeDenied)
		return
	}

	nodes, err := h.db.GetNodesByWorkflowID(ctx, workflowID)

	workflowIter := validate.NewWorkflowNodeIter(userWorkflow.StartNodeID, nodes)
	deleteNodeMaterial, err := workflowIter.GetDeleteNodeMaterial(nodeID)
	if err != nil {
		err = fmt.Errorf("get deleteNodeMaterial error: %w", err)
		return
	}

	// delete node: first change transitions based on type of the node, then delete related nodes.
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		preNode := deleteNodeMaterial.PreviousNode
		needUpdatePreviousNode := true

		// If the node is trigger node, should update workflow.StartNodeID to empty.
		if isDeletingStartNode {
			err = h.db.UpdateWorkflowStartNodeID(ctx, workflowID, "")
			if err != nil {
				err = fmt.Errorf("updating workflow startNodeID: %w", err)
				return
			}

			// remove trigger record.
			err = h.db.DeleteTriggersByWorkflowID(ctx, workflowID)
			if err != nil {
				err = fmt.Errorf("deleting triggers: %w", err)
				return
			}
		}

		// change transitions
		// 1. first child node of foreach node
		// 2. first child node of switch node
		// 3. first node of the workflow, we should not change the transition of its previous node
		// 4. normal node
		if preNode.Class == validate.ForeachClass && preNode.Transition != currentNode.ID { // 1. make sure current node is inside foreach node
			var (
				forEachNodeStruct validate.LoopFromListNode
				forEachNodeMap    map[string]any
			)

			err = trans.MapToStruct(preNode.Data.InputFields, &forEachNodeStruct)
			if err != nil {
				err = fmt.Errorf("foreach node input field error: %w", err)
				return
			}
			forEachNodeStruct.Transition = currentNode.Transition

			forEachNodeMap, err = trans.StructToMap(forEachNodeStruct)
			if err != nil {
				err = fmt.Errorf("foreach node struct error: %w", err)
				return
			}
			preNode.Data.InputFields = forEachNodeMap
		} else if preNode.Class == validate.SwitchClass { // 2.
			var (
				switchNodeStruct validate.SwitchLogicNode
				switchNodeMap    map[string]any
			)

			err = trans.MapToStruct(preNode.Data.InputFields, &switchNodeStruct)
			if err != nil {
				err = fmt.Errorf("switch node input field error: %w", err)
				return
			}
			for i := range switchNodeStruct.Paths {
				if switchNodeStruct.Paths[i].Transition == nodeID {
					switchNodeStruct.Paths[i].Transition = currentNode.Transition
					break
				}
			}

			switchNodeMap, err = trans.StructToMap(switchNodeStruct)
			if err != nil {
				err = fmt.Errorf("switch node struct error: %w", err)
				return
			}
			preNode.Data.InputFields = switchNodeMap
		} else if preNode.ID == "" { // 3.
			needUpdatePreviousNode = false
		} else { // 4.
			preNode.Transition = currentNode.Transition
		}

		if needUpdatePreviousNode {
			err = tx.UpdateNodeByID(ctx, &preNode)
			if err != nil {
				return
			}
		}

		// delete related node
		err = tx.DeleteNodeByIDs(ctx, deleteNodeMaterial.DeleteNodeIDChain)
		if err != nil {
			err = fmt.Errorf("delete related node error: %w", err)
			return
		}

		return
	})
	if err != nil {
		return
	}

	OK(c, nil)
}

// GetNodeTestPageData get data in test page.
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   nodeId  path string true "node id"
// @Success 200 {object} apiserver.R{data=response.GetNodeTestPageDataResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/nodes/{nodeId}/testPageData [get]
func (h *APIHandler) GetNodeTestPageData(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		workflowID = c.Params.ByName("id")
		nodeID     = c.Params.ByName("nodeId")
		resp       = response.GetNodeTestPageDataResp{}
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = errBizInvalidWorkflowID
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	node, err := h.db.GetNodeByID(ctx, nodeID)
	if err != nil {
		err = fmt.Errorf("query node: %w", err)
		return
	}

	trigger, err := h.db.GetTriggerByNodeID(ctx, node.ID)
	if err != nil {
		err = fmt.Errorf("query trigger: %w", err)
		return
	}

	data := map[string]any{}
	// TODO(Sword): how to abstract this logic?
	// Every different trigger maybe has different webhook url, wait refactor! Hard code here temporarily!
	if node.Class == validate.CustomWebhookClass {
		data["webhookURL"] = h.serverHost.WebhookFullURL(fmt.Sprintf("hooks/%s", trigger.ID))
	}

	resp.InputFields = data
	OK(c, resp)
}
