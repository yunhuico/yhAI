package apiserver

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"strconv"
)

// GetUserTour returns current user tour.
// @Summary Return current user tour
// @Produce json
// @Success 200 {object} apiserver.R{data=response.GetUserToursResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tour [get]
func (h *APIHandler) GetUserTour(c *gin.Context) {
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

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}

	tours, err := h.db.GetUserToursByUserID(ctx, objectOwner.OwnerID)
	if err != nil {
		err = fmt.Errorf("querying user tours: %w", err)
		return
	}

	mapTours := response.GetUserToursInMap(tours)

	OK(c, mapTours)
}

// DeleteUserTour delete user tour by user id.
// @Summary delete user tour by user id
// @Produce json
// @Param   id  path int true "user id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tour/{id} [delete]
func (h *APIHandler) DeleteUserTour(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		id  = c.Param("id")
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}

	if id == "" {
		err = fmt.Errorf("invalid user id")
		return
	}

	userID, err := strconv.Atoi(id)
	if err != nil {
		err = fmt.Errorf("invalid user id")
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		tours, err := tx.GetUserToursByUserID(ctx, userID)
		if err != nil {
			err = fmt.Errorf("quering user tour: %w", err)
			return
		}
		ids := make([]string, 0, 4)
		for _, tour := range tours {
			ids = append(ids, tour.ID)
		}

		err = tx.DeleteTourStepWithTourIDs(ctx, ids...)
		if err != nil {
			err = fmt.Errorf("deleting user tour step: %w", err)
			return
		}

		err = tx.DeleteToursByUserID(ctx, objectOwner.OwnerID)
		if err != nil {
			err = fmt.Errorf("deleting user tours: %w", err)
			return
		}
		return
	})

	OK(c, nil)
}

// CreateUserTour create tour for current user.
// @Summary create tour
// @Produce json
// @Param   body body payload.CreateUserTourReq true "the payload"
// @Success 200 {object} apiserver.R{data=response.GetUserToursResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tour [post]
func (h *APIHandler) CreateUserTour(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.CreateUserTourReq
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

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}

	var tourCfg model.TourConfig
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		tourCfg, err = tx.GetTourConfigWithStepsByPath(ctx, req.Path)
		if err != nil {
			err = fmt.Errorf("query tour config by path: %w", err)
			return
		}

		userTour := model.UserTour{
			UserID: objectOwner.OwnerID,
			Path:   req.Path,
		}
		err = tx.InsertUserTour(ctx, &userTour)
		if err != nil {
			err = fmt.Errorf("inserting user tour: %w", err)
			return
		}

		for i := range tourCfg.Steps {
			userStep := model.UserTourStep{
				Path:        req.Path,
				Title:       tourCfg.Steps[i].Title,
				Description: tourCfg.Steps[i].Description,
				Idx:         tourCfg.Steps[i].Index,
				TourID:      userTour.ID,
				ElementID:   tourCfg.Steps[i].ElementID,
				Status:      tourCfg.Steps[i].Status,
			}
			err = tx.InsertTourStep(ctx, &userStep)
			if err != nil {
				err = fmt.Errorf("inserting user tour step: %w", err)
				return
			}
			if userStep.Idx == 1 {
				_, err = tx.UpdateUserTourByID(ctx, userTour.ID, model.TourStatusProcessing, userStep.ID)
				if err != nil {
					err = fmt.Errorf("updating user tour: %w", err)
					return
				}
			}
		}
		return
	})
	if err != nil {
		err = fmt.Errorf("database transaction aborted: %w", err)
		return
	}
	tours, err := h.db.GetUserToursByUserID(ctx, objectOwner.OwnerID)
	if err != nil {
		err = fmt.Errorf("querying user tours: %w", err)
		return
	}
	mapTours := response.GetUserToursInMap(tours)

	OK(c, mapTours)
}

// UpdateUserTour update tour for current user.
// @Summary update tour
// @Produce json
// @Param   body body payload.UpdateUserTourReq true "the payload"
// @Success 200 {object} apiserver.R{data=response.GetUserToursResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tour [put]
func (h *APIHandler) UpdateUserTour(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.UpdateUserTourReq
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}

	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}

	var tour model.UserTour
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		tour, err = tx.GetUserTourByID(ctx, req.ID)
		if err != nil {
			err = fmt.Errorf("query tour with id %s: %w", req.ID, err)
			return err
		}

		switch req.Status {
		case "skipped":
			tour.Status = model.TourStatusSkipped
		case "complete":
			tour.Status = model.TourStatusComplete
		default:
			tour.Status = model.TourStatusProcessing
		}

		tour.CurStepID = req.Step
		_, err = tx.UpdateUserTourByID(ctx, req.ID, tour.Status, tour.CurStepID)
		if err != nil {
			err = fmt.Errorf("update user tour: %w", err)
			return err
		}

		return nil
	})

	if err != nil {
		err = fmt.Errorf("database transaction aborted: %w", err)
		return
	}
	tours, err := h.db.GetUserToursByUserID(ctx, objectOwner.OwnerID)
	if err != nil {
		err = fmt.Errorf("querying user tours: %w", err)
		return
	}
	mapTours := response.GetUserToursInMap(tours)

	OK(c, mapTours)
}

// CreateTourConfig create tour configuration.
// @Summary create tour configuration
// @Produce json
// @Param   body body payload.CreateTourConfigReq true "the payload"
// @Success 200 {object} apiserver.R{data=model.TourConfig}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tourconfig [post]
func (h *APIHandler) CreateTourConfig(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.CreateTourConfigReq
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

	tourConfig := model.TourConfig{
		Path:        req.Path,
		Description: req.Description,
		Title:       req.Title,
	}

	err = h.db.InsertTourConfig(ctx, &tourConfig)
	if err != nil {
		err = fmt.Errorf("creating tour config: %w", err)
		return
	}

	OK(c, tourConfig)
}

// CreateTourScopeStep create tour step configuration.
// @Summary create tour step configuration
// @Produce json
// @Param   body body payload.CreateTourStepConfigReq true "the payload"
// @Param   id  path string true "tour config id"
// @Success 200 {object} apiserver.R{data=model.TourPathStepConfig}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tourconfig/{id}/step [post]
func (h *APIHandler) CreateTourScopeStep(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.CreateTourStepConfigReq
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

	stepConfig := model.TourPathStepConfig{
		TourConfigID: req.TourConfigID,
		Title:        req.Title,
		Description:  req.Description,
		Index:        req.Index,
		Status:       req.Status,
		ElementID:    req.ElementID,
	}

	err = h.db.InsertTourScopeStep(ctx, &stepConfig)
	if err != nil {
		err = fmt.Errorf("creating tour config: %w", err)
		return
	}
	OK(c, stepConfig)
}

// GetTourConfigs returns tour configuration.
// @Summary Return tour configuration
// @Produce json
// @Success 200 {object} apiserver.R{data=[]model.TourConfig}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tourconfig [get]
func (h *APIHandler) GetTourConfigs(c *gin.Context) {
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

	tourConfigs, err := h.db.GetTourConfigWithSteps(ctx)
	if err != nil {
		err = fmt.Errorf("getting tour config: %w", err)
		return
	}
	OK(c, tourConfigs)
}

// UpdateTourConfig update tour configuration.
// @Summary update tour step configuration
// @Produce json
// @Param   body body payload.UpdateTourConfigReq true "the payload"
// @Param   id  path string true "tour config id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tourconfig/{id}/config [put]
func (h *APIHandler) UpdateTourConfig(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.UpdateTourConfigReq
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

	tourConfig := model.TourConfig{
		ID:          req.ID,
		Path:        req.Path,
		Title:       req.Title,
		Description: req.Description,
	}

	err = h.db.UpdateTourConfig(ctx, &tourConfig)
	if err != nil {
		err = fmt.Errorf("updating tour config: %w", err)
		return
	}
	OK(c, nil)
}

// UpdateTourStepConfigElementID update tour configuration.
// @Summary update tour step configuration
// @Produce json
// @Param   body body payload.UpdateTourStepConfigReq true "the payload"
// @Param   id  path string true "tour step id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tourconfig/{id}/step [put]
func (h *APIHandler) UpdateTourStepConfigElementID(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.UpdateTourStepConfigReq
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

	_, err = h.db.UpdateTourScopeStepElementID(ctx, req.ID, req.ElementID)
	if err != nil {
		err = fmt.Errorf("updating tour config: %w", err)
		return
	}
	OK(c, nil)
}

// DeleteTourConfigByID delete tour configuration by id.
// @Summary delete tour configuration by id
// @Produce json
// @Param   id  path string true "tour config id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/tourconfig/{id}/config [delete]
func (h *APIHandler) DeleteTourConfigByID(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		id  = c.Param("id")
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.DeleteTourStepConfigByConfigID(ctx, id)
		if err != nil {
			err = fmt.Errorf("deleting tour step config: %w", err)
			return
		}

		err = tx.DeleteTourConfigByID(ctx, id)
		if err != nil {
			err = fmt.Errorf("deleting tour config: %w", err)
			return
		}
		return
	})

	if err != nil {
		err = fmt.Errorf("deleting tour config: %w", err)
		return
	}
	OK(c, nil)
}
