package apiserver

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	workflowCmd "jihulab.com/jihulab/ultrafox/ultrafox/pkg/cmd/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/featureflag"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/validator"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

const (
	defaultIconNumber = 5
)

// ListWorkflow list workflows by page.
// @Summary list workflows by page
// @Produce json
// @Param   iconNumber query int 4 "the number of workflow using adapter icon"
// @Param   limit query int 20 "the count of workflows per page"
// @Param   offset query int 0 "the offset of the select from"
// @Param   orgId query int false "id of the owner organization"
// @Success 200 {object} apiserver.R{data=response.ListWorkflowResponse}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows [get]
func (h *APIHandler) ListWorkflow(c *gin.Context) {
	type Req struct {
		IconNumber int `form:"iconNumber"`
	}

	var (
		err        error
		ctx        = c.Request.Context()
		req        Req
		iconNumber = 4
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	limit, offset, _, err := extractPageParameters(c)
	if err != nil {
		err = fmt.Errorf("parsing paging parms: %w", err)
		return
	}

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}

	err = c.ShouldBindQuery(&req)
	if err != nil {
		err = fmt.Errorf("binding query: %w", err)
		return
	}
	if req.IconNumber > 0 {
		iconNumber = req.IconNumber
	}

	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, objectOwner, permission.WorkflowRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	workflows, count, err := h.db.ListWorkflowsWithNodesByOwner(ctx, objectOwner, limit, offset)
	if err != nil {
		err = fmt.Errorf("querying workflows: %w", err)
		return
	}

	workflowsDist := h.formatWorkflows(ctx, workflows, iconNumber)
	OK(c, &response.ListWorkflowResponse{
		Total:     count,
		Workflows: workflowsDist,
	})
}

// CreateWorkflow create workflow.
// @Summary create workflow
// @Produce json
// @Param   body body payload.EditWorkflowReq true "the payload"
// @Param   orgId query int false "id of the owner organization"
// @Success 200 {object} apiserver.R{data=response.ResourceCreatedResponse}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows [post]
func (h *APIHandler) CreateWorkflow(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.EditWorkflowReq
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
		return
	}
	if err = validator.Validate(req); err != nil {
		return
	}

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, objectOwner, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	userWorkflow := &model.Workflow{
		OwnerRef:    objectOwner,
		Name:        req.Name,
		Description: req.Description,
		Status:      model.WorkflowStatusDisabled,
	}
	err = h.db.InsertWorkflow(ctx, userWorkflow)
	if err != nil {
		err = fmt.Errorf("inserting workflow: %w", err)
		return
	}

	OK(c, response.ResourceCreatedResponse{
		ID: userWorkflow.ID,
	})
}

// UpdateWorkflow update workflow
// @Summary update workflow basic fields
// @Produce json
// @Param   body body payload.EditWorkflowReq true "the payload"
// @Param   id  path string true "workflow id"
// @Param   orgId query int false "id of the owner organization"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id} [put]
func (h *APIHandler) UpdateWorkflow(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		workflowID = c.Param("id")
		req        payload.EditWorkflowReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowID == "" {
		err = errors.New("invalid workflow id")
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}
	if err = validator.Validate(req); err != nil {
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow by id %q: %w", workflowID, err)
		return
	}
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	workflowToUpdate := &model.Workflow{
		ID:          workflowID,
		Name:        req.Name,
		Description: req.Description,
	}
	err = h.db.UpdateWorkflowByID(ctx, workflowToUpdate, "name", "description")
	if err != nil {
		err = fmt.Errorf("updating workflow by id %q: %w", workflowToUpdate.ID, err)
		return
	}

	OK(c, nil)
}

// DeleteWorkflow delete workflow
// @Summary delete workflow by id
// @Produce json
// @Param   id  path string true "workflow id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id} [delete]
func (h *APIHandler) DeleteWorkflow(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		workflowID = c.Param("id")
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
		return
	}
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, userWorkflow.OwnerRef, permission.WorkflowDelete)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	if userWorkflow.Status == model.WorkflowStatusEnabled {
		err = errDeleteWorkflowDenied
		return
	}

	deleteCommand := workflowCmd.NewDeleteCommand(h.db.Operator, h.triggerRegistry)
	err = deleteCommand.DeleteMetaData(ctx, userWorkflow.ID)
	if err != nil {
		err = fmt.Errorf("delete workflow: %w", err)
		return
	}

	OK(c, nil)
}

// getWorkflowIcons
// get icons for workflow as given number
func getWorkflowIcons(userWorkflow model.WorkflowWithNodes, iconNumber int) []string {
	adapterManager := adapter.GetAdapterManager()
	adapterClassSet := set.Set[string]{}
	adapterClasses := make([]string, 0, iconNumber)
	for _, node := range userWorkflow.Nodes {
		if len(adapterClassSet) == iconNumber {
			break
		}
		if !adapterClassSet.Has(node.Data.MetaData.AdapterClass) {
			adapterClasses = append(adapterClasses, node.Data.MetaData.AdapterClass)
		}
		adapterClassSet.Add(node.Data.MetaData.AdapterClass)
	}
	return adapterManager.GetIconsByClass(adapterClasses)
}

// formatWorkflows...
// current affects: add icons to every workflow and set nodes to nil.
func (h *APIHandler) formatWorkflows(ctx context.Context, workflows []model.WorkflowWithNodes, iconNumber int) []*response.WorkflowWithIcons {
	workflowsDist := make([]*response.WorkflowWithIcons, len(workflows))

	for i, workflow := range workflows {
		icons, err := getSortedNodesIcons(workflow.StartNodeID, workflow.Nodes, iconNumber)
		if err != nil {
			h.logger.For(ctx).Error("get sorted nodes icons error", log.String("workflowID", workflow.ID), log.ErrField(err))
		}

		workflow.Nodes = nil
		workflowsDist[i] = &response.WorkflowWithIcons{
			Workflow: workflow.Workflow,
			Icons:    icons,
		}
	}
	return workflowsDist
}

func getSortedNodesIcons(startNodeID string, nodes model.Nodes, iconNumber int) (icons []string, err error) {
	adapterManager := adapter.GetAdapterManager()
	adapterClasses := []string{}

	iter := validate.NewWorkflowNodeIter(startNodeID, nodes)
	err = iter.Loop(func(node model.Node) (end bool) {
		if len(adapterClasses) == iconNumber {
			return true
		}

		adapterClasses = append(adapterClasses, node.GetAdapterClass())
		return
	})

	icons = adapterManager.GetIconsByClass(adapterClasses)
	return
}

// ListAllWorkflowLog list all workflow log by page
// @Produce json
// @Param   limit query int false "default is 20"
// @Param   offset query int false "default is 0"
// @Param   orgId query int false "id of the owner organization"
// @Param	workflowName query string false "name that target workflow contains"
// @Param	startTime query string false "startTime, format: YYYY-mm-dd HH:mm:ss"
// @Param	endTime	query string false "endTime, format: YYYY-mm-dd HH:mm:ss"
// @Param	status query string false "choose one: scheduled, running, paused, failed, completed. Leave empty or missing to search for all status."
// @Param	isAsc query bool false "false(default): order by desc; true: order by asc"
// @Param	iconNumber query string false "iconNumber, default is 5"
// @Success 200 {object} apiserver.R{data=response.ListWorkflowLogResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/log [get]
func (h *APIHandler) ListAllWorkflowLog(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.SearchLogReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	limit, offset, _, err := extractPageParameters(c)
	if err != nil {
		err = fmt.Errorf("parsing paging parms: %w", err)
		return
	}

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}

	err = c.ShouldBindQuery(&req)
	if err != nil {
		err = fmt.Errorf("binding query: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, objectOwner, permission.WorkflowLogRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	workflowInstances, count, err := h.db.SearchWorkflowInstances(ctx, model.SearchLogOptions{
		OwnerRef:     objectOwner,
		WorkflowName: req.WorkflowName,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Status:       req.Status,
		IsAsc:        req.IsAsc,
	}, limit, offset)

	if err != nil {
		err = fmt.Errorf("listing workflow instances: %w", err)
		return
	}

	if len(workflowInstances) == 0 {
		resp := &response.ListWorkflowLogResp{
			Total:             count,
			WorkflowInstances: nil,
		}
		OK(c, resp)
		return
	}

	workflowIDSet := set.Set[string]{}
	for _, workflowInstance := range workflowInstances {
		workflowIDSet.Add(workflowInstance.WorkflowID)
	}

	workflowIDList := workflowIDSet.All()
	workflows, err := h.db.GetWorkflowsWithNodesByIDs(ctx, workflowIDList...)
	if err != nil {
		err = fmt.Errorf("listing workflows: %w", err)
		return
	}
	workflowsMap := make(map[string]*model.WorkflowWithNodes, len(workflows))
	for i := range workflows {
		workflowsMap[workflows[i].ID] = &workflows[i]
	}

	resp := &response.ListWorkflowLogResp{
		Total:             count,
		WorkflowInstances: make([]response.WorkflowInstanceWithWorkflow, len(workflowInstances)),
	}

	for i, workflowInstance := range workflowInstances {
		resp.WorkflowInstances[i].WorkflowInstance = workflowInstance
		resp.WorkflowInstances[i].Workflow = &workflowsMap[workflowInstance.WorkflowID].Workflow
		resp.WorkflowInstances[i].Icons = getWorkflowIcons(*workflowsMap[workflowInstance.WorkflowID], req.IconNumber)
	}

	OK(c, resp)
}

// EnableWorkflow enable workflow, set status to active
// @Summary Enable a workflow
// @Description Enable a workflow by id
// @Accept json
// @Produce json
// @Param  id  path string true "workflow id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/enable [post]
func (h *APIHandler) EnableWorkflow(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	workflowID := c.Params.ByName("id")
	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	userWorkflow, err := h.db.GetWorkflowWithNodesByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("quryinng workflow by id %q: %w", workflowID, err)
		return
	}

	report := validate.ValidateWorkflow(userWorkflow)
	if report.ExistsFatal() {
		err = fmt.Errorf("validating workflow: %s", report.Error())
		_ = c.Error(err)
		err = errBizWorkflowCheckFail
		return
	}

	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowSwitch)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	if userWorkflow.Status == model.WorkflowStatusEnabled {
		// nothing to do, relax
		OK(c, nil)
		return
	}

	var allNodesPassTest bool
	allNodesPassTest, err = h.calculateWorkflowNodesPassTest(ctx, userWorkflow.StartNodeID, userWorkflow.Nodes)
	if err != nil {
		err = fmt.Errorf("calculating workflow's nodes pass-test: %w", err)
		return
	}

	if !allNodesPassTest {
		err = errNodesShouldTestBeforeActive
		return
	}

	triggerNode, err := h.db.GetTriggerNodeByWorkflowID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("getting trigger node: %w", err)
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		var trigger model.Trigger
		trigger, err = tx.GetOrCreateTrigger(ctx, workflowID, &triggerNode)
		if err != nil {
			err = fmt.Errorf("getting or creating trigger: %w", err)
			return
		}

		triggerWithNodeSession := model.TriggerWithNode{
			Trigger: trigger,
			Node:    &triggerNode,
		}
		err = h.triggerRegistry.EnableTrigger(ctx, tx, &triggerWithNodeSession)
		if err != nil {
			err = fmt.Errorf("creating trigger on node %q: %w", triggerNode.ID, err)
			return
		}

		err = tx.UpdateWorkflowStatus(ctx, workflowID, model.WorkflowStatusEnabled)
		if err != nil {
			err = fmt.Errorf("updating workflow status: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, nil)
}

// DisableWorkflow disable a workflow
// @Summary disables a workflow
// @Description disables a workflow by id
// @Accept json
// @Produce json
// @Param  id  path string true "workflow id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/disable [post]
func (h *APIHandler) DisableWorkflow(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	workflowID := c.Params.ByName("id")
	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow by id %q: %w", workflowID, err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowSwitch)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	if userWorkflow.Status != model.WorkflowStatusEnabled {
		OK(c, nil)
		return
	}

	err = h.disableWorkflow(ctx, userWorkflow.ID)
	if err != nil {
		err = fmt.Errorf("disabling workflow: %w", err)
		return
	}

	OK(c, nil)
}

func (h *APIHandler) disableWorkflow(ctx context.Context, workflowID string) (err error) {
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		deleteCommand := workflowCmd.NewDeleteCommand(tx, h.triggerRegistry)
		err = deleteCommand.DeleteTriggerResource(ctx, workflowID)
		if err != nil {
			err = fmt.Errorf("deleting triggerr resource: %w", err)
			return
		}

		// set workflow status to inactive
		err = tx.UpdateWorkflowStatus(ctx, workflowID, model.WorkflowStatusDisabled)
		if err != nil {
			err = fmt.Errorf("updating workflow status: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}
	return
}

// RunWorkflow run a workflow manually
// @Summary run a workflow
// @Description run a workflow by id
// @ID runWorkflow
// @Accept json
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   body  body payload.RunWorkflowReq true "request body"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/run [post]
func (h *APIHandler) RunWorkflow(c *gin.Context) {
	var (
		err       error
		ctx       = c.Request.Context()
		startedAt = time.Now()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	var req payload.RunWorkflowReq
	err = c.ShouldBindJSON(&req)
	if err != nil || req.NodeID == "" {
		err = errBizInvalidRequestPayload
		return
	}

	workflowID := c.Params.ByName("id")
	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	userWorkflow, err := h.db.GetWorkflowWithNodesCredentialByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow with nodes by id %q: %w", workflowID, err)
		return
	}
	currentUserID := getSession(c).UserID
	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowRun)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	const APIRunMaxStep = 100
	workflowCtx, err := workflow.NewWorkflowContext(ctx, workflow.ContextOpt{
		DB:                   h.db,
		Cache:                h.cache,
		WorkflowWithNodes:    userWorkflow,
		Cipher:               h.cipher,
		ServerHost:           h.serverHost,
		MaxSteps:             APIRunMaxStep,
		MailSender:           h.mailSender,
		PassportVendorLookup: h.passportVendorLookup,
	})
	if err != nil {
		err = fmt.Errorf("building workflow context: %w", err)
		return
	}

	report := validate.ValidateWorkflow(model.WorkflowWithNodes{
		Workflow: userWorkflow.Workflow,
		Nodes:    userWorkflow.Nodes.GetNodes(),
	})
	if report.ExistsFatal() {
		err = fmt.Errorf("validating workflow: %s", report.LogString())
		return
	}

	if !req.UseExternalInput {
		workflowCtx.UseInputFields()
	}
	result := workflowCtx.Run(req.NodeID, req.Input)

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		// bookkeeping the execution
		workflowInstance := &model.WorkflowInstance{
			WorkflowID:  userWorkflow.ID,
			Status:      result.Status,
			StartNodeID: req.NodeID,
			FailNodeID:  result.FailNodeID,
			StartTime:   startedAt,
			DurationMs:  int(time.Since(startedAt).Milliseconds()),
			Error:       result.Error(),
		}
		err = h.db.InsertWorkflowInstance(ctx, workflowInstance)
		if err != nil {
			err = fmt.Errorf("insert workflow instance: %w", err)
			return
		}
		for _, instanceNode := range result.InstanceNodes {
			instanceNode.WorkflowInstanceID = workflowInstance.ID
		}
		err = h.db.BulkInsertWorkflowInstanceNodes(ctx, result.InstanceNodes)
		if err != nil {
			err = fmt.Errorf("bulk insert workflow instance nodes: %w", err)
			return
		}
		return
	})

	if err != nil {
		err = fmt.Errorf("db tx error after run workflow: %w", err)
		return
	}

	if result.Err != nil {
		err = fmt.Errorf("running workflow: %w", result.Err)
		return
	}

	OK(c, nil)
}

// GetWorkflow get workflow detail
// @Summary get workflow detail
// @Produce json
// @Param   id  path string true "workflow id"
// @Success 200 {object} apiserver.R{data=response.GetWorkflowResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id} [get]
func (h *APIHandler) GetWorkflow(c *gin.Context) {
	var err error
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	workflowID := c.Params.ByName("id")
	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	var workflowDetail response.WorkflowDetail
	workflowDetail, err = h.getWorkflowDetail(c, workflowID)
	if err != nil {
		return
	}

	OK(c, response.GetWorkflowResp{
		Workflow: workflowDetail,
	})
}

func (h *APIHandler) getWorkflowDetail(c *gin.Context, workflowID string) (detail response.WorkflowDetail, err error) {
	ctx := c.Request.Context()
	userWorkflow, err := h.db.GetWorkflowWithNodesByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	var allNodesPassTest bool
	allNodesPassTest, err = h.calculateWorkflowNodesPassTest(ctx, userWorkflow.StartNodeID, userWorkflow.Nodes)
	if err != nil {
		err = fmt.Errorf("calculating workflow's nodes pass-test: %w", err)
		return
	}

	detail = response.WorkflowDetail{
		WorkflowWithNodes: &userWorkflow,
		AllNodesPassTest:  allNodesPassTest,
	}
	return
}

// GetWorkflowExtra get workflow extra information.
// @Summary get workflow extra information
// @Produce json
// @Param   id  path string true "workflow id"
// @Success 200 {object} apiserver.R{data=response.GetWorkflowExtraResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/extra [get]
func (h *APIHandler) GetWorkflowExtra(c *gin.Context) {
	var err error
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	workflowID := c.Params.ByName("id")
	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	var workflowDetail response.WorkflowDetail
	workflowDetail, err = h.getWorkflowDetail(c, workflowID)
	if err != nil {
		return
	}

	OK(c, response.GetWorkflowExtraResp{
		AllNodesPassTest: workflowDetail.AllNodesPassTest,
	})
}

func (h *APIHandler) calculateWorkflowNodesPassTest(ctx context.Context, nodeID string, nodes model.Nodes) (allNodesPassTest bool, err error) {
	if !featureflag.IsEnabled(ctx, featureflag.WorkflowEnableCheck, featureflag.ContextData{}) {
		allNodesPassTest = true
		return
	}

	needfulTestNodeIDSet := set.Set[string]{}
	iter := validate.NewWorkflowNodeIter(nodeID, nodes)
	err = iter.Loop(func(node model.Node) (end bool) {
		if node.Class != validate.SwitchClass {
			needfulTestNodeIDSet.Add(node.ID)
		}
		return
	})
	if err != nil {
		err = fmt.Errorf("getting all needful test nodes: %w", err)
		return
	}
	for _, node := range nodes {
		if node.TestSuccessedOrSkipped() {
			needfulTestNodeIDSet.Delete(node.ID)
		}
	}
	allNodesPassTest = needfulTestNodeIDSet.Len() == 0
	return
}

// ListWorkflowLog list workflow log by page
// @Produce json
// @Param   id  path string true "workflow id"
// @Param   iconNumber query int 4 "the number of workflow using adapter icon"
// @Param   limit query int false "default is 20"
// @Param   offset query int false "default is 0"
// @Success 200 {object} apiserver.R{data=response.ListWorkflowLogResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/{id}/log [get]
func (h *APIHandler) ListWorkflowLog(c *gin.Context) {
	var (
		err           error
		ctx           = c.Request.Context()
		iconNumber    = defaultIconNumber
		iconNumberStr = c.DefaultQuery("iconNumber", "5")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if v, err := strconv.Atoi(iconNumberStr); err == nil {
		iconNumber = v
	}

	workflowID := c.Params.ByName("id")
	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	limit, offset, _, err := extractPageParameters(c)
	if err != nil {
		return
	}

	userWorkflow, err := h.db.GetWorkflowWithNodesByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow details: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowLogRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	workflowInstances, count, err := h.db.ListWorkflowInstancesByWorkflowID(ctx, workflowID, limit, offset)
	if err != nil {
		err = fmt.Errorf("querying workflow instances by id %q", workflowID)
		return
	}

	icons := getWorkflowIcons(userWorkflow, iconNumber)
	output := make([]response.WorkflowInstanceWithWorkflow, len(workflowInstances))
	for i, instance := range workflowInstances {
		output[i].WorkflowInstance = instance
		output[i].Icons = icons
	}

	OK(c, &response.ListWorkflowLogResp{
		Total:             count,
		WorkflowInstances: output,
	})
}

// ApplyWorkflowYaml applies workflow yaml
// @Produce json
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/apply [put]
func (h *APIHandler) ApplyWorkflowYaml(c *gin.Context) {
	var (
		ctx = c.Request.Context()
		err error
		req model.WorkflowWithNodes
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	decoder := yaml.NewDecoder(c.Request.Body)
	err = decoder.Decode(&req)
	if err != nil {
		err = fmt.Errorf("decoding workflow yaml: %w", err)
		return
	}

	req.OwnerRef = model.OwnerRef{
		OwnerType: model.OwnerTypeUser,
		OwnerID:   getSession(c).UserID,
	}
	var workflowID string
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		workflowID, err = workflow.Apply(ctx, h.triggerRegistry, &req, tx)
		if err != nil {
			err = fmt.Errorf("applying workflow: %w", err)
			return
		}
		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, workflowID)
}
