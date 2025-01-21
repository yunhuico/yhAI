package apiserver

import (
	"database/sql"
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/share"
)

// ExportWorkflow export a workflow to yaml file
// @Summary Export a workflow to yaml file
// @Description export a workflow by id
// @Accept json
// @Produce json
// @Param  id  path string true "workflow id"
// @Success 200 {object} apiserver.R{data=response.ExportWorkflowResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/{id}/export [post]
func (h *APIHandler) ExportWorkflow(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

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

	workflowYaml, err := share.Export(ctx, &userWorkflow, h.db)
	if err != nil {
		err = fmt.Errorf("export workflow yaml: %w", err)
		return
	}

	OK(c, response.ExportWorkflowResp{WorkflowYaml: workflowYaml})
}

// ImportWorkflow import a workflow yaml
// @Summary import a workflow yaml
// @Produce json
// @Param   orgId query int false "id of the owner organization"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/import [post]
func (h *APIHandler) ImportWorkflow(c *gin.Context) {
	const (
		MaxFileSize = 20 << 20 // 20 MB
		fileFormKey = "file"
	)

	var (
		ctx               = c.Request.Context()
		err               error
		workflowWithNodes model.WorkflowWithNodes
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	err = c.Request.ParseMultipartForm(MaxFileSize)
	if err != nil {
		err = fmt.Errorf("parsing file from form: %w", err)
		return
	}

	yamlFile, _, err := c.Request.FormFile(fileFormKey)
	yamlBytes, err := io.ReadAll(yamlFile)
	if err != nil {
		err = fmt.Errorf("reading file: %w", err)
		return
	}

	err = yamlFile.Close()
	if err != nil {
		err = fmt.Errorf("closing yaml file: %w", err)
		return
	}

	err = yaml.Unmarshal(yamlBytes, &workflowWithNodes)
	if err != nil {
		err = fmt.Errorf("unmarshaling workflow yaml: %w", err)
		return
	}

	err = share.SanitizeImport(&workflowWithNodes)
	if err != nil {
		err = fmt.Errorf("importing workflow: %w", err)
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

	workflowWithNodes.OwnerRef = objectOwner
	var workflowID string
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		workflowID, err = workflow.Apply(ctx, h.triggerRegistry, &workflowWithNodes, tx)
		if err != nil {
			err = fmt.Errorf("applying workflow: %w", err)
			return
		}
		return
	})

	OK(c, workflowID)
}

// EnableOrCreateWorkflowShareLink Export a workflow as a URL, if already exists, it turns on the link
// @Summary Export a workflow as a URL
// @Accept json
// @Produce json
// @Param  id  path string true "workflow id"
// @Success 200 {object} apiserver.R{data=model.WorkflowShareLink}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/{id}/exportUrl [post]
func (h *APIHandler) EnableOrCreateWorkflowShareLink(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowShareLinkRead, permission.WorkflowShareLinkWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	link, err := h.db.GetWorkflowShareLinkByWorkflowID(ctx, workflowID)
	if !errors.Is(err, sql.ErrNoRows) && err != nil {
		err = fmt.Errorf("querying workflow link: %w", err)
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		link, err = h.db.InsertWorkflowShareLink(ctx, workflowID)
		if err != nil {
			err = fmt.Errorf("inserting workflow link: %w", err)
			return
		}
	} else {
		err = h.db.UpdateWorkflowShareLinkStatusByID(ctx, link.ID, model.StatusOn)
		if err != nil {
			err = fmt.Errorf("updating workflow link status: %w", err)
			return
		}
	}

	OK(c, link)
}

// ResetWorkflowShareLink resets the sharing workflow URL
// @Summary resets the sharing workflow URL
// @Accept json
// @Produce json
// @Param  id  path string true "workflow id"
// @Success 200 {object} apiserver.R{data=model.WorkflowShareLink}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/{id}/exportUrl [put]
func (h *APIHandler) ResetWorkflowShareLink(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowShareLinkRead, permission.WorkflowShareLinkWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	var link model.WorkflowShareLink
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.DeleteWorkflowShareLinkByWorkflowID(ctx, workflowID)
		if err != nil {
			err = fmt.Errorf("deleting workflow link: %w", err)
			return
		}
		link, err = tx.InsertWorkflowShareLink(ctx, workflowID)
		if err != nil {
			err = fmt.Errorf("generating workflow link: %w", err)
			return
		}
		return
	})

	OK(c, link)
}

// DisableWorkflowShareLink disables the sharing workflow link
// @Summary disables the sharing workflow link
// @Accept json
// @Produce json
// @Param  id  path string true "workflow id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/{id}/exportUrl [delete]
func (h *APIHandler) DisableWorkflowShareLink(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowShareLinkWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	link, err := h.db.GetWorkflowShareLinkByWorkflowID(ctx, workflowID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizWorkflowShareLinkNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying workflow link: %w", err)
		return
	}

	if link.Status == model.StatusOff {
		OK(c, nil)
		return
	}

	err = h.db.UpdateWorkflowShareLinkStatusByID(ctx, link.ID, model.StatusOff)
	if err != nil {
		err = fmt.Errorf("updating workflow link status: %w", err)
		return
	}

	OK(c, nil)
}

// GetWorkflowShareLink gets the workflow link
// @Summary gets the workflow link
// @Accept json
// @Produce json
// @Param  id  path string true "workflow id"
// @Success 200 {object} apiserver.R{data=model.WorkflowShareLink}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/{id}/exportUrl [get]
func (h *APIHandler) GetWorkflowShareLink(c *gin.Context) {
	var (
		ctx        = c.Request.Context()
		err        error
		workflowID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	userWorkflow, err := h.db.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying workflow: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, userWorkflow.OwnerRef, permission.WorkflowShareLinkRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	link, err := h.db.GetWorkflowShareLinkByWorkflowID(ctx, workflowID)

	if errors.Is(err, sql.ErrNoRows) {
		OK(c, nil)
		return
	}

	if err != nil {
		err = fmt.Errorf("querying workflow link: %w", err)
		return
	}

	OK(c, link)
}

// BrowseWorkflowShareLink a user browses workflow URL
// @Summary a user browses workflow URL
// @Accept json
// @Produce json
// @Param  id  path string true "workflow link id"
// @Success 200 {object} apiserver.R{data=response.BrowseWorkflowShareLinkResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/importUrl/{id} [get]
func (h *APIHandler) BrowseWorkflowShareLink(c *gin.Context) {
	var (
		ctx                 = c.Request.Context()
		err                 error
		workflowShareLinkID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowShareLinkID == "" {
		err = errBizInvalidWorkflowID
		return
	}

	link, err := h.db.GetWorkflowShareLinkByID(ctx, workflowShareLinkID)
	if errors.Is(err, sql.ErrNoRows) || link.Status == model.StatusOff {
		err = errBizWorkflowShareLinkNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying workflow link: %w", err)
		return
	}

	workflowWithNodes, err := h.db.GetWorkflowWithNodesByID(ctx, link.WorkflowID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizWorkflowNotFound
		return
	}

	authorName, authorAvatar, err := share.GetAuthorNameAvatar(ctx, workflowWithNodes.OwnerType, workflowWithNodes.OwnerID, h.db)
	if err != nil {
		err = fmt.Errorf("querying user name or organization name: %w", err)
		return
	}

	icons := getWorkflowIcons(workflowWithNodes, defaultIconNumber)
	OK(c, response.BrowseWorkflowShareLinkResp{
		WorkflowWithNodes: workflowWithNodes,
		Icons:             icons,
		Annotations: share.Annotations{
			Author:    authorName,
			AvatarURL: authorAvatar,
			CreatedAt: link.CreatedAt,
		},
	})
}

// ValidateWorkflowShareLink checks whether the workflow link is valid
// @Summary checks whether the workflow link is valid
// @Accept json
// @Produce json
// @Param  id  path string true "workflow link id"
// @Success 200 {object} apiserver.R{data=model.WorkflowShareLink}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/importUrl/{id}/validate [get]
func (h *APIHandler) ValidateWorkflowShareLink(c *gin.Context) {
	var (
		ctx                 = c.Request.Context()
		err                 error
		workflowShareLinkID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowShareLinkID == "" {
		err = errBizInvalidWorkflowShareLinkID
		return
	}

	link, err := h.db.GetWorkflowShareLinkByID(ctx, workflowShareLinkID)
	if errors.Is(err, sql.ErrNoRows) || link.Status == model.StatusOff {
		err = errBizWorkflowShareLinkNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying workflow link: %w", err)
		return
	}

	OK(c, link)
}

// ImportWorkflowShareLink import a workflow from a URL
// @Summary Import a workflow from a URL
// @Accept json
// @Produce json
// @Param  id  path string true "workflow link id"
// @Param   orgId query int false "id of the owner organization"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/workflows/share/importUrl/{id}/accept [post]
func (h *APIHandler) ImportWorkflowShareLink(c *gin.Context) {
	var (
		ctx                 = c.Request.Context()
		err                 error
		workflowShareLinkID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if workflowShareLinkID == "" {
		err = errBizInvalidWorkflowShareLinkID
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

	link, err := h.db.GetWorkflowShareLinkByID(ctx, workflowShareLinkID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizWorkflowShareLinkNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying workflow link: %w", err)
		return
	}

	workflowWithNodes, err := h.db.GetWorkflowWithNodesByID(ctx, link.WorkflowID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizWorkflowNotFound
		return
	}

	err = share.SanitizeImport(&workflowWithNodes)
	if err != nil {
		err = fmt.Errorf("importing workflow: %w", err)
		return
	}

	workflowWithNodes.OwnerRef = objectOwner
	var workflowID string
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		workflowID, err = workflow.Apply(ctx, h.triggerRegistry, &workflowWithNodes, tx)
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
