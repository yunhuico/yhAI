package apiserver

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"github.com/gin-gonic/gin"
)

// AdminListTemplateTags list all template tags with all fields
//
// @Summary list all template tags with all fields
// @Produce json
// @Success 200 {object} apiserver.R{data=[]model.TemplateTag}
// @Failure 400 {object} apiserver.R
// @Router /api/admin/templates/tags [get]
func (h *APIHandler) AdminListTemplateTags(c *gin.Context) {
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

	tags, err := h.db.ListTemplateTags(ctx)
	if err != nil {
		err = fmt.Errorf("querying tags: %w", err)
		return
	}

	OK(c, tags)
}

// AdminCreateTemplateTag create a new template tag
//
// @Summary create a new template tag
// @Produce json
// @Param   body body model.EditableTemplateTag true "the payload"
// @Success 200 {object} apiserver.R{data=model.TemplateTag}
// @Failure 400 {object} apiserver.R
// @Router /api/admin/templates/tags [post]
func (h *APIHandler) AdminCreateTemplateTag(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req model.EditableTemplateTag
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

	if !model.IsValidTemplateTagRole(req.Role) {
		err = fmt.Errorf("unexpected template tag role %q", req.Role)
		return
	}

	req.Label = strings.TrimSpace(req.Label)

	tag := model.TemplateTag{
		EditableTemplateTag: req,
	}
	err = h.db.InsertTemplateTag(ctx, &tag)
	if err != nil {
		err = fmt.Errorf("inserting template tag: %w", err)
		return
	}

	OK(c, &tag)
}

// AdminUpdateTemplateTag change the specified template tag
//
// @Summary change the specified template tag
// @Produce json
// @Param   id  path int true "template tag id"
// @Param   body body model.EditableTemplateTag true "the payload"
// @Success 200 {object} apiserver.R{data=model.TemplateTag}
// @Failure 400 {object} apiserver.R
// @Router /api/admin/templates/tags/{id} [put]
func (h *APIHandler) AdminUpdateTemplateTag(c *gin.Context) {
	var (
		err      error
		ctx      = c.Request.Context()
		tagID, _ = strconv.Atoi(c.Param("id"))
		req      model.EditableTemplateTag
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if tagID == 0 {
		err = errors.New("invalid tag id")
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}

	if !model.IsValidTemplateTagRole(req.Role) {
		err = fmt.Errorf("unexpected template tag role %q", req.Role)
		return
	}

	req.Label = strings.TrimSpace(req.Label)

	tag := model.TemplateTag{
		ID:                  tagID,
		EditableTemplateTag: req,
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.UpdateTemplateTag(ctx, &tag)
		if err != nil {
			err = fmt.Errorf("updating template tag: %w", err)
			return
		}

		err = tx.RefreshTemplateRedundantFieldsByTagID(ctx, tag.ID)
		if err != nil {
			err = fmt.Errorf("refreshing tmeplate redundant fields: %w", err)
			return
		}
		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, &tag)
}

// AdminDeleteTemplateTag delete the specified template tag.
//
// @Summary delete the specified template tag.
// @Produce json
// @Param   id  path int true "template tag id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/admin/templates/tags/{id} [delete]
func (h *APIHandler) AdminDeleteTemplateTag(c *gin.Context) {
	var (
		err      error
		ctx      = c.Request.Context()
		tagID, _ = strconv.Atoi(c.Param("id"))
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if tagID == 0 {
		err = errors.New("invalid tag id")
		return
	}

	_, err = h.db.GetTemplateTagByID(ctx, tagID)
	if errors.Is(err, sql.ErrNoRows) {
		// relax
		OK(c, nil)
		return
	}
	if err != nil {
		err = fmt.Errorf("querying template tag: %w", err)
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.DeleteTemplateTagByID(ctx, tagID)
		if err != nil {
			err = fmt.Errorf("deleting template tag: %w", err)
			return
		}

		err = tx.RefreshTemplateRedundantFieldsByTagID(ctx, tagID)
		if err != nil {
			err = fmt.Errorf("refreshing tmeplate redundant fields: %w", err)
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
