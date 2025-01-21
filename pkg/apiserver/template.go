package apiserver

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/share"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

type SearchTemplatesReq struct {
	model.SearchTemplateParam
}

type SearchTemplatesResult struct {
	Icons []string `json:"icons"`
	model.SearchTemplateHit
}

type SearchTemplatesResp struct {
	Total      int                       `json:"total"`
	Categories []model.SimpleTemplateTag `json:"categories"`
	Templates  []SearchTemplatesResult   `json:"templates"`
}

// SearchTemplates search templates for template list. No auth is needed.
//
// @Summary search templates for template list. No auth is needed.
// @Produce json
// @Param   body body SearchTemplatesReq true "the payload"
// @Success 200 {object} apiserver.R{data=SearchTemplatesResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates/search [post]
func (h *APIHandler) SearchTemplates(c *gin.Context) {
	var (
		err  error
		ctx  = c.Request.Context()
		req  SearchTemplatesReq
		resp SearchTemplatesResp
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

	resp.Categories, err = h.db.GetTemplateCategories(ctx)
	if err != nil {
		err = fmt.Errorf("querying for template categories: %w", err)
		return
	}
	hits, count, err := h.db.SearchTemplates(ctx, req.SearchTemplateParam)
	if err != nil {
		err = fmt.Errorf("searching for templates: %w", err)
		return
	}
	resp.Total = count

	adapterManager := adapter.GetAdapterManager()
	resp.Templates = make([]SearchTemplatesResult, len(hits))
	for i := range resp.Templates {
		resp.Templates[i] = SearchTemplatesResult{
			Icons:             adapterManager.GetIconsByClass(hits[i].RelatedAdapters),
			SearchTemplateHit: hits[i],
		}
	}

	OK(c, &resp)
}

type GetTemplateTagsResp struct {
	Categories []model.SimpleTemplateTag `json:"categories"`
}

// GetTemplateTags get template tags, categories, etc
//
// @Summary get template tags, categories, etc
// @Produce json
// @Success 200 {object} apiserver.R{data=GetTemplateTagsResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates/tags [get]
func (h *APIHandler) GetTemplateTags(c *gin.Context) {
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

	categories, err := h.db.GetTemplateCategories(ctx)
	if err != nil {
		err = fmt.Errorf("querying template categories: %w", err)
		return
	}

	resp := GetTemplateTagsResp{
		Categories: categories,
	}

	OK(c, &resp)
}

type GetTemplateResult struct {
	Icons []string `json:"icons"`
	model.Template
}

// GetTemplate get template detail by id. No auth is needed.
//
// @Summary get template detail by id. No auth is needed.
// @Produce json
// @Param   id  path string true "template id"
// @Success 200 {object} apiserver.R{data=GetTemplateResult}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates/{id} [get]
func (h *APIHandler) GetTemplate(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		templateID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	var resp GetTemplateResult

	resp.Template, err = h.db.GetTemplateByID(ctx, templateID, model.TemplateStatusOK)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.New("the template does not exist, or is not ready to be public")
		return
	}
	if err != nil {
		err = fmt.Errorf("querying template: %w", err)
		return
	}

	adapterManager := adapter.GetAdapterManager()
	resp.Icons = adapterManager.GetIconsByClass(resp.RelatedAdapters)

	OK(c, &resp)
}

// UseTemplate use template by id
//
// @Summary use template by id
// @Produce json
// @Param   orgId query int false "id of the owner organization"
// @Param   id  path string true "template id"
// @Success 200 {object} apiserver.R{data=string}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates/{id}/use [post]
func (h *APIHandler) UseTemplate(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		templateID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	destOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding dest owner: %w", err)
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, destOwner, permission.WorkflowUpsert)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		return
	}

	template, err := h.db.GetTemplateByID(ctx, templateID, model.TemplateStatusOK)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.New("the template does not exist, or is not ready to be public")
		return
	}
	if err != nil {
		err = fmt.Errorf("querying template: %w", err)
		return
	}

	workflowWithNodes := template.Content
	err = share.SanitizeImport(workflowWithNodes)
	if err != nil {
		err = fmt.Errorf("sanitizing import: %w", err)
		return
	}

	workflowWithNodes.OwnerRef = destOwner
	workflowWithNodes.Name = template.Title
	workflowWithNodes.Description = template.Description

	var workflowID string
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		workflowID, err = workflow.Apply(ctx, h.triggerRegistry, workflowWithNodes, tx)
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

type ListTemplatesByOwnerResp struct {
	Total     int              `json:"total"`
	Templates []model.Template `json:"templates"`
}

// ListTemplatesByOwner list user/org's templates, including all status
//
// @Summary list user/org's templates, including all status
// @Produce json
// @Param   orgId query int false "id of the owner organization"
// @Success 200 {object} apiserver.R{data=ListTemplatesByOwnerResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates/byOwner [get]
func (h *APIHandler) ListTemplatesByOwner(c *gin.Context) {
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

	owner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding owner: %w", err)
		return
	}
	limit, offset, _, err := extractPageParameters(c)

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, owner, permission.TemplateRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		return
	}

	templates, count, err := h.db.ListTemplatesByOwner(ctx, owner, limit, offset)
	if err != nil {
		err = fmt.Errorf("listing templates by owner: %w", err)
		return
	}

	resp := ListTemplatesByOwnerResp{
		Total:     count,
		Templates: templates,
	}

	OK(c, &resp)
}

type PublishTemplateReq struct {
	// which workflow does this template originate?
	WorkflowID string `json:"workflowId" binding:"required"`
	// Categories to connect with
	CategoryIDs []int `json:"categoryIDs" binding:"required,min=1"`

	// Optional below

	// Tags to connect with
	TagIDs []int `json:"tagIds"`
	// title of the item, leave blank to infer from the workflow name
	Title string `json:"title"`
	// short description for list, defaults to description.
	Brief string `json:"brief" binding:"max=100"`
	// long description in the template detail page, defaults to the workflow's description.
	Description string `json:"description" binding:"max=8000"`
	// the greater, the former, defaults to 0
	Weight int `json:"weight,omitempty" binding:"min=0"`
}

type PublishTemplateResp struct {
	TemplateID string               `json:"templateId"`
	Status     model.TemplateStatus `json:"status"`
}

// PublishTemplate publish a template, based on an existed workflow
//
// @Summary publish a template, based on an existed workflow
// @Produce json
// @Param   orgId query int false "id of the owner organization"
// @Param   body body PublishTemplateReq true "the payload"
// @Success 200 {object} apiserver.R{data=PublishTemplateResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates [post]
func (h *APIHandler) PublishTemplate(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req PublishTemplateReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	destOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding dest owner: %w", err)
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, destOwner, permission.TemplateWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		return
	}

	userWorkflow, err := h.db.GetWorkflowWithNodesByID(ctx, req.WorkflowID)
	if errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("%s %d does not has access to workflow %s, or the workflow does not exist", destOwner.OwnerType, destOwner.OwnerID, req.WorkflowID)
		return
	}
	if err != nil {
		err = fmt.Errorf("querying specified workflow: %w", err)
		return
	}
	// ensure the workflow's owner is destOwner
	if userWorkflow.OwnerType != destOwner.OwnerType ||
		userWorkflow.OwnerID != destOwner.OwnerID {
		err = fmt.Errorf("%s %d does not has access to workflow %s, or the workflow does not exist", destOwner.OwnerType, destOwner.OwnerID, req.WorkflowID)
		return
	}

	if len(userWorkflow.Nodes) == 0 {
		err = fmt.Errorf("workflow %s has 0 node", userWorkflow.ID)
		return
	}

	// handling of defaults
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		req.Title = strings.TrimSpace(userWorkflow.Name)
	}
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		req.Description = strings.TrimSpace(userWorkflow.Description)
	}
	req.Brief = strings.TrimSpace(req.Brief)
	if req.Brief == "" {
		req.Brief = req.Description
	}

	relatedAdapters, err := quickRelatedAdapters(userWorkflow.StartNodeID, userWorkflow.Nodes)
	if err != nil {
		err = fmt.Errorf("resolving workflow related adapters: %w", err)
		return
	}

	oldWorkflowID := userWorkflow.ID

	err = share.SanitizeImport(&userWorkflow)
	if err != nil {
		err = fmt.Errorf("sanitizing the workflow: %w", err)
		return
	}
	userWorkflow.OwnerRef = model.OwnerRef{}
	userWorkflow.Status = model.WorkflowStatusDisabled
	// avoid leaking sensitive data
	userWorkflow.Name = ""
	userWorkflow.Description = ""
	userWorkflow.Version = ""

	template := model.Template{
		Status:           model.TemplateStatusPendingReview,
		OwnerRef:         destOwner,
		Weight:           req.Weight,
		Title:            req.Title,
		Brief:            req.Brief,
		Description:      req.Description,
		RelatedAdapters:  relatedAdapters,
		SourceWorkflowID: oldWorkflowID,
		Content:          &userWorkflow,

		// the following fields are left empty for RefreshTemplateRedundantFields

		AuthorName:     "",
		AuthorAvatar:   "",
		CategoryIDs:    nil,
		CategoryLabels: nil,
		TagIDs:         nil,
		TagLabels:      nil,
	}

	// For Jihulab user, the template does not to be reviewed.
	user, err := h.db.GetUserByID(ctx, getSession(c).UserID)
	if err != nil {
		err = fmt.Errorf("querying for current user: %w", err)
		return
	}
	if isUserInsider(user.Email) {
		template.Status = model.TemplateStatusOK
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.InsertTemplate(ctx, &template)
		if err != nil {
			err = fmt.Errorf("inserting template: %w", err)
			return
		}
		err = tx.InsertTemplateTagRelByTemplateID(ctx, template.ID, append(req.CategoryIDs, req.TagIDs...)...) // LOL, wow, so many dots
		if err != nil {
			err = fmt.Errorf("inserting template tag relations: %w", err)
			return
		}
		err = tx.RefreshTemplateRedundantFields(ctx, template.ID)
		if err != nil {
			err = fmt.Errorf("refreshing template redundant fields: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	resp := PublishTemplateResp{
		TemplateID: template.ID,
		Status:     template.Status,
	}

	OK(c, &resp)
}

func quickRelatedAdapters(startNodeID string, nodes []model.Node) (adapterClass []string, err error) {
	size := len(nodes)
	if size > 16 {
		size = 16
	}

	adapterClass = make([]string, 0, size)
	seen := make(set.Set[string], size)

	iter := validate.NewWorkflowNodeIter(startNodeID, nodes)
	err = iter.Loop(func(node model.Node) (end bool) {
		class := node.GetAdapterClass()
		if seen.Has(class) {
			return
		}
		seen.Add(class)

		adapterClass = append(adapterClass, class)
		return
	})
	if err != nil {
		err = fmt.Errorf("iterating nodes: %w", err)
		return
	}

	return
}

type UpdateTemplateReq struct {
	// Categories to connect with
	CategoryIDs []int `json:"categoryIDs" binding:"required,min=1"`
	// Tags to connect with
	TagIDs []int `json:"tagIds"`
	// title of the item, leave blank to infer from the workflow name
	Title string `json:"title" binding:"required"`
	// short description for list, defaults to description.
	Brief string `json:"brief" binding:"required,max=100"`
	// long description in the template detail page, defaults to the workflow's description.
	Description string `json:"description" binding:"required,max=8000"`
	// the greater, the former, defaults to 0
	Weight int `json:"weight,omitempty" binding:"min=0"`

	// Optional below

	// Which workflow does this template originate?
	// The template content is refreshed if provided.
	WorkflowID string `json:"workflowId"`
}

type UpdateTemplateResp struct {
	TemplateID string               `json:"templateId"`
	Status     model.TemplateStatus `json:"status"`
}

// UpdateTemplate update a template by id, every field is required except workflowId
//
// @Summary update a template by id, every field is required except workflowId
// @Produce json
// @Param   body body UpdateTemplateReq true "the payload"
// @Param   id  path string true "template id"
// @Success 200 {object} apiserver.R{data=UpdateTemplateResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates/{id} [patch]
func (h *APIHandler) UpdateTemplate(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		req        UpdateTemplateReq
		templateID = c.Param("id")
		changed    = []string{"title", "brief", "description", "weight", "status"}
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	template, err := h.db.GetTemplateByID(ctx, templateID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.New("template does not exist, or you don't have required permission")
		return
	}
	if err != nil {
		err = fmt.Errorf("querying template: %w", err)
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, template.OwnerRef, permission.TemplateWrite)
	if err != nil {
		_ = c.Error(fmt.Errorf("ensuring permission: %w", err))
		err = errors.New("template does not exist, or you don't have required permission")
		return
	}

	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}

	// any update needs a review
	template.Status = model.TemplateStatusPendingReview
	template.Title = strings.TrimSpace(req.Title)
	template.Brief = strings.TrimSpace(req.Brief)
	template.Description = strings.TrimSpace(req.Description)
	template.Weight = req.Weight

	// For Jihulab user, the template does not to be reviewed.
	user, err := h.db.GetUserByID(ctx, getSession(c).UserID)
	if err != nil {
		err = fmt.Errorf("querying for current user: %w", err)
		return
	}
	if isUserInsider(user.Email) {
		template.Status = model.TemplateStatusOK
	}

	if req.WorkflowID != "" {
		var userWorkflow model.WorkflowWithNodes

		// Template content needs to be updated
		userWorkflow, err = h.db.GetWorkflowWithNodesByID(ctx, req.WorkflowID)
		if errors.Is(err, sql.ErrNoRows) {
			err = fmt.Errorf("%s %d does not has access to workflow %s, or the workflow does not exist", template.OwnerType, template.OwnerID, req.WorkflowID)
			return
		}
		if err != nil {
			err = fmt.Errorf("querying specified workflow: %w", err)
			return
		}
		// ensure the workflow's owner is template's owner
		if userWorkflow.OwnerType != template.OwnerType ||
			userWorkflow.OwnerID != template.OwnerID {
			err = fmt.Errorf("%s %d does not has access to workflow %s, or the workflow does not exist", template.OwnerType, template.OwnerID, req.WorkflowID)
			return
		}

		if len(userWorkflow.Nodes) == 0 {
			err = fmt.Errorf("workflow %s has 0 node", userWorkflow.ID)
			return
		}

		template.RelatedAdapters, err = quickRelatedAdapters(userWorkflow.StartNodeID, userWorkflow.Nodes)
		if err != nil {
			err = fmt.Errorf("resolving workflow related adapters: %w", err)
			return
		}

		template.SourceWorkflowID = userWorkflow.ID

		err = share.SanitizeImport(&userWorkflow)
		if err != nil {
			err = fmt.Errorf("sanitizing the workflow: %w", err)
			return
		}
		userWorkflow.OwnerRef = model.OwnerRef{}
		userWorkflow.Status = model.WorkflowStatusDisabled
		// avoid leaking sensitive data
		userWorkflow.Name = ""
		userWorkflow.Description = ""
		userWorkflow.Version = ""

		template.Content = &userWorkflow
		changed = append(changed, "related_adapters", "source_workflow_id", "content")
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.UpdateTemplate(ctx, &template, changed...)
		if err != nil {
			err = fmt.Errorf("updating template: %w", err)
			return
		}
		err = tx.DeleteTemplateTagRelByTemplateID(ctx, templateID)
		if err != nil {
			err = fmt.Errorf("deleting template tag relation: %w", err)
			return
		}
		err = tx.InsertTemplateTagRelByTemplateID(ctx, template.ID, append(req.CategoryIDs, req.TagIDs...)...) // LOL, wow, so many dots
		if err != nil {
			err = fmt.Errorf("inserting template tag relations: %w", err)
			return
		}
		err = tx.RefreshTemplateRedundantFields(ctx, template.ID)
		if err != nil {
			err = fmt.Errorf("refreshing template redundant fields: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	resp := UpdateTemplateResp{
		TemplateID: template.ID,
		Status:     template.Status,
	}

	OK(c, &resp)
}

// DeleteTemplate delete a template by id
//
// @Summary delete a template by id
// @Produce json
// @Param   id  path string true "template id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/templates/{id} [delete]
func (h *APIHandler) DeleteTemplate(c *gin.Context) {
	var (
		err        error
		ctx        = c.Request.Context()
		templateID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	template, err := h.db.GetTemplateByID(ctx, templateID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.New("template does not exist, or you don't have required permission")
		return
	}
	if err != nil {
		err = fmt.Errorf("querying template: %w", err)
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, template.OwnerRef, permission.TemplateWrite)
	if err != nil {
		_ = c.Error(fmt.Errorf("ensuring permission: %w", err))
		err = errors.New("template does not exist, or you don't have required permission")
		return
	}

	err = h.db.DeleteTemplate(ctx, templateID)
	if err != nil {
		err = fmt.Errorf("deleting template: %w", err)
		return
	}

	OK(c, nil)
}
