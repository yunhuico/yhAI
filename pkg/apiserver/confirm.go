package apiserver // GetConfirm fetch confirm by id
import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

// GetConfirm fetches confirm by id
// @Produce json
// @Param   id  path string true "the confirm record id"
// @Success 200 {object} apiserver.R{data=response.GetConfirmResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/confirm/{id} [get]
func (h *APIHandler) GetConfirm(c *gin.Context) {
	var (
		err       error
		ctx       = c.Request.Context()
		confirmID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if confirmID == "" {
		err = errors.New("confirm id is required")
		return
	}
	confirmWithWorkflow, err := h.db.GetConfirmWithWorkflowByID(ctx, confirmID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errInvalidConfirmation
		return
	}
	if err != nil {
		err = fmt.Errorf("querying confirm: %w", err)
		return
	}

	userID := getSession(c).UserID
	err = h.enforcer.EnsurePermissions(ctx, userID, confirmWithWorkflow.Workflow.OwnerRef, permission.WorkflowExecutionAuthorization)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	// is the user in the confirmer list?
	var isConfirmer bool
	for _, confirmerID := range confirmWithWorkflow.Confirmers {
		if userID == confirmerID {
			isConfirmer = true
			break
		}
	}
	if !isConfirmer {
		_ = c.Error(fmt.Errorf("user is not in the confirmer list of confirm %q", confirmWithWorkflow.ID))
		err = errNoPermissionError
		return
	}

	workflowWithNodes, err := h.db.GetWorkflowWithNodesByID(ctx, confirmWithWorkflow.WorkflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow with nodes: %w", err)
		return
	}

	OK(c, response.GetConfirmResp{
		Expired:         confirmWithWorkflow.IsExpired(),
		WorkflowEnabled: confirmWithWorkflow.Workflow.Status == model.WorkflowStatusEnabled,
		Confirm:         confirmWithWorkflow.Confirm,
		Workflow:        workflowWithNodes,
	})
}

// DecideConfirm user makes confirm decision
// @Produce json
// @Param   id  path string true "the confirm record id"
// @Param   body body payload.DecideConfirmReq true "the payload"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/confirm/{id}/decision [post]
func (h *APIHandler) DecideConfirm(c *gin.Context) {
	var (
		err       error
		ctx       = c.Request.Context()
		confirmID = c.Param("id")
		req       payload.DecideConfirmReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if confirmID == "" {
		err = errors.New("confirm id is required")
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}

	var newConfirmStatus model.ConfirmStatus
	switch req.Decision {
	case workflow.ConfirmDecisionApproved:
		newConfirmStatus = model.ConfirmStatusApproved
	case workflow.ConfirmDecisionDeclined:
		newConfirmStatus = model.ConfirmStatusDeclined
	default:
		err = fmt.Errorf("unexpected decision %q", req.Decision)
		return
	}

	confirmWithWorkflow, err := h.db.GetConfirmWithWorkflowByID(ctx, confirmID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errInvalidConfirmation
		return
	}
	if err != nil {
		err = fmt.Errorf("querying confirm: %w", err)
		return
	}

	userID := getSession(c).UserID
	err = h.enforcer.EnsurePermissions(ctx, userID, confirmWithWorkflow.Workflow.OwnerRef, permission.WorkflowExecutionAuthorization)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	// is the user in the confirmer list?
	var isConfirmer bool
	for _, confirmerID := range confirmWithWorkflow.Confirmers {
		if userID == confirmerID {
			isConfirmer = true
			break
		}
	}
	if !isConfirmer {
		_ = c.Error(fmt.Errorf("user is not in the confirmer list of confirm %q", confirmWithWorkflow.ID))
		err = errNoPermissionError
		return
	}

	if confirmWithWorkflow.Status != model.ConfirmStatusWaiting {
		_ = c.Error(fmt.Errorf("confirm status is not %q, got %q", model.ConfirmStatusWaiting, confirmWithWorkflow.Status))
		err = errInvalidConfirmation
		return
	}
	if confirmWithWorkflow.IsExpired() {
		_ = c.Error(fmt.Errorf("confirm is already expired by %s", confirmWithWorkflow.Confirm.ExpiredAt.Format(time.RFC3339)))
		err = errInvalidConfirmation
		return
	}
	if confirmWithWorkflow.Workflow.Status != model.WorkflowStatusEnabled {
		_ = c.Error(fmt.Errorf("workflow status is %d, want %d", confirmWithWorkflow.Workflow.Status, model.WorkflowStatusEnabled))
		err = errInvalidConfirmation
		return
	}

	err = h.db.UpdateWaitingConfirmStatusByID(ctx, confirmID, newConfirmStatus, userID)
	if err != nil {
		err = fmt.Errorf("updating confirm status: %w", err)
		return
	}

	userWork := work.Work{
		ID:          confirmWithWorkflow.WorkflowInstanceID,
		WorkflowID:  confirmWithWorkflow.WorkflowID,
		Resume:      true,
		StartNodeID: confirmWithWorkflow.NodeID,
	}
	err = h.workProducer.Produce(ctx, &userWork)
	if err != nil {
		err = fmt.Errorf("producing work: %w", err)
		return
	}

	OK(c, nil)
}
