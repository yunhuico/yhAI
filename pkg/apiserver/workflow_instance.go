package apiserver

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
)

// GetWorkflowInstanceDetail get workflow instance log detail
// @Produce json
// @Param   id  path string true "workflow instance id"
// @Success 200 {object} apiserver.R{data=response.DetailedWorkflowInstanceResp} "0:completed 1:failed 2:running 3:scheduled"
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflowInstances/{id}/detail [get]
func (h *APIHandler) GetWorkflowInstanceDetail(c *gin.Context) {
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

	workflowInstID := c.Params.ByName("id")
	if workflowInstID == "" {
		err = errBizInvalidWorkflowInstanceID
		return
	}

	inst, err := h.db.GetWorkflowInstanceWithNodesByID(ctx, workflowInstID)
	if err != nil {
		err = fmt.Errorf("querying workflow instance by id %q: %w", workflowInstID, err)
		_ = c.Error(err)
		err = errBizReadDatabase
		return
	}

	workflow, err := h.db.GetWorkflowWithNodesByID(ctx, inst.WorkflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow %q: %w", inst.WorkflowID, err)
		_ = c.Error(err)
		err = errBizReadDatabase
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, workflow.OwnerRef, permission.WorkflowLogRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	resp, err := response.GetDetailedWorkflowInstanceResp(inst, workflow)
	if err != nil {
		err = fmt.Errorf("composing workflow instance detail: %w", err)
		return
	}
	OK(c, resp)
}

// DeleteWorkflowInstance delete a run instance of workflow
// @Param   id  path string true "workflow instance id"
// @Success 200 {object}  apiserver.R
// @Failure 400 {object}  apiserver.R
// @Router /api/v1/workflowInstances/{id} [delete]
func (h *APIHandler) DeleteWorkflowInstance(c *gin.Context) {
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

	workflowInstID := c.Params.ByName("id")
	if workflowInstID == "" {
		err = errBizInvalidWorkflowInstanceID
		return
	}

	inst, err := h.db.GetWorkflowInstanceWithNodesByID(ctx, workflowInstID)
	if err != nil {
		err = fmt.Errorf("querying workflow instance by id %q: %w", workflowInstID, err)
		_ = c.Error(err)
		err = errBizReadDatabase
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, inst.WorkflowID)
	if err != nil {
		err = fmt.Errorf("qurying workflow by id: %w", err)
		_ = c.Error(err)
		err = errBizReadDatabase
		return
	}

	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowLogWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	err = h.db.DeleteWorkflowInstanceByID(ctx, workflowInstID)
	if err != nil {
		err = fmt.Errorf("deleting workflow instance by id: %w", err)
		return
	}

	OK(c, nil)
}
