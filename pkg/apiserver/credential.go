package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"

	"github.com/gin-gonic/gin"
	xoauth2 "golang.org/x/oauth2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/oauth2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/service"
)

// ListCredential get credential by page.
// @Produce json
// @Param   orgId query int false "id of the owner organization"
// @Param   adapterClass query string false "filter by adapter class"
// @Param   limit query int 20 "the count of workflows per page"
// @Param   offset query int 0 "the offset of the select from"
// @Success 200 {object} apiserver.R{data=response.ListCredentialResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/credentials [get]
func (h *APIHandler) ListCredential(c *gin.Context) {
	var (
		err           error
		ctx           = c.Request.Context()
		limit, offset int
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	limit, offset, _, err = extractPageParameters(c)
	adapterClass := c.Query("adapterClass")

	objectOwner, err := bindObjectOwner(c)
	if err != nil {
		err = fmt.Errorf("binding object owner: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, objectOwner, permission.CredentialRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	var (
		count       int
		credentials []model.Credential
	)
	if adapterClass == "" {
		credentials, count, err = h.db.ListCredentialsByOwner(ctx, objectOwner, limit, offset)
	} else {
		credentials, count, err = h.db.ListCredentialsByOwnerAndAdapterClass(ctx, objectOwner, adapterClass, limit, offset)
	}
	if err != nil {
		err = fmt.Errorf("querying credentials: %w", err)
		return
	}

	OK(c, response.ListCredentialResp{
		Total:       count,
		Credentials: credentials,
	})
}

// GetCredential get credential by id
// @Summary get credential by id
// @Produce json
// @Success 200 {object} apiserver.R{data=response.GetCredentialResp}
// @Failure 400 {object} apiserver.R
// @Param   id  path string true "credential id"
// @Router /api/v1/credentials/{id} [get]
func (h *APIHandler) GetCredential(c *gin.Context) {
	var (
		ctx          = c.Request.Context()
		err          error
		credentialID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	credential, err := h.db.GetCredentialByID(ctx, credentialID)
	if err != nil {
		err = fmt.Errorf("querying credential: %w", err)
		return
	}
	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, credential.OwnerRef, permission.CredentialRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	var inputFields map[string]string
	// all official don't support to check details.
	// and all OAuth credential only support see and edit their names.
	if credential.OfficialName == "" && credential.Type != model.CredentialTypeOAuth {
		err = h.decryptor.Unmarshal(credential.Data, &inputFields)
		if err != nil {
			return
		}

		var credentialForm *adapter.CredentialForm
		credentialForm, err = getCredentialForm(credential.AdapterClass, string(credential.Type))
		if err != nil {
			err = fmt.Errorf("get credential form: %w", err)
			return
		}

		inputFields = credentialForm.Masker.Mask(inputFields)
	}

	OK(c, &response.GetCredentialResp{
		ID: credential.ID,
		EditableCredential: model.EditableCredential{
			Name:         credential.Name,
			AdapterClass: credential.AdapterClass,
			Type:         credential.Type,
			OfficialName: credential.OfficialName,
		},
		Status:      credential.Status,
		InputFields: inputFields,
		CreatedAt:   credential.CreatedAt,
		UpdatedAt:   credential.UpdatedAt,
	})
}

const skipTestCredentialKey = "__skipTesting__"

// CreateCredential
// @Summary create credential
// @Produce json
// @Param   body body payload.EditCredentialReq true "the payload"
// @Param   orgId query int false "id of the owner organization"
// @Success 200 {object} apiserver.R{data=response.ResourceCreatedResponse}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/credentials [post]
func (h *APIHandler) CreateCredential(c *gin.Context) {
	const timeout = 5 * time.Second

	var (
		err         error
		ctx, cancel = context.WithTimeout(c.Request.Context(), timeout)
		skipTesting = c.Query(skipTestCredentialKey) == "1"
	)
	defer func() {
		cancel()
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
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, objectOwner, permission.CredentialWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	req := &payload.EditCredentialReq{}
	if err = c.ShouldBindJSON(req); err != nil {
		err = errBizInvalidRequestPayload
		return
	}

	credential, err := h.assembleCredential(c, req)
	if err != nil {
		return
	}
	credential.OwnerRef = objectOwner
	credential.UpdatedAt = time.Now()

	if !skipTesting {
		err = h.testCredential(ctx, req.AdapterClass, req.Type, req.InputFields)
		if err != nil {
			_ = c.Error(err)
			err = errTestCredentialFail
			return
		}
	}

	err = h.db.InsertCredential(ctx, credential)
	if err != nil {
		err = fmt.Errorf("inserting credential: %w", err)
		return
	}

	OK(c, response.ResourceCreatedResponse{ID: credential.ID})
}

func (h *APIHandler) testCredential(ctx context.Context, adapterClass string, credentialType model.CredentialType, inputFields map[string]string) (err error) {
	adapterManager := adapter.GetAdapterManager()
	meta := adapterManager.LookupAdapter(adapterClass)
	if meta == nil {
		err = fmt.Errorf("unknown adapter %q", adapterClass)
		return
	}

	err = meta.TestCredential(ctx, credentialType, inputFields)
	if err != nil {
		err = fmt.Errorf("testing credential: %w", err)
		return
	}
	return
}

func (h *APIHandler) assembleCredential(c *gin.Context, req *payload.EditCredentialReq) (credential *model.Credential, err error) {
	ctx := c.Request.Context()
	if err = req.Validate(ctx); err != nil {
		err = fmt.Errorf("validating credential: %w", err)
		return
	}

	credential, err = req.Normalize()
	if err != nil {
		h.logger.For(ctx).Error("create credential normalize payload failed", log.ErrField(err))
		return
	}

	// todo(sword): temporary code, delete this after official credentials done.
	if req.AdapterClass == "ultrafox/slackCorpBot" {
		req.OfficialName = "dingtalkCorpBot"
	}

	var encryptedData []byte
	if req.OfficialName != "" {
		officialCredential, ok := h.getOfficialCredential(req.OfficialName)
		if !ok {
			err = fmt.Errorf("unknown official credential %q", req.OfficialName)
			return
		}
		credential.OfficialName = req.OfficialName
		encryptedData, err = h.decryptor.Marshal(officialCredential.GetMergedData(req.InputFields))
	} else {
		encryptedData, err = h.decryptor.Marshal(req.InputFields)
	}

	if err != nil {
		return
	}

	credential.Data = encryptedData

	return
}

func getCredentialForm(adapterClass, credentialType string) (form *adapter.CredentialForm, err error) {
	adapterManager := adapter.GetAdapterManager()
	meta := adapterManager.LookupAdapter(adapterClass)
	if meta == nil {
		err = fmt.Errorf("adapter %q not found", adapterClass)
		return
	}
	return meta.GetCredentialForm(credentialType)
}

// UpdateCredential
// @Summary update credential by id
// @Produce json
// @Param   body body payload.EditCredentialReq true "the payload"
// @Param   id  path string true "credential id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/credentials/{id} [put]
func (h *APIHandler) UpdateCredential(c *gin.Context) {
	const timeout = 5 * time.Second

	var (
		ctx, cancel  = context.WithTimeout(c.Request.Context(), timeout)
		err          error
		credentialID = c.Param("id")
		skipTesting  = c.Query(skipTestCredentialKey) == "1"
	)
	defer func() {
		cancel()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	req := &payload.EditCredentialReq{}
	if err = c.ShouldBindJSON(req); err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	if err = req.Validate(ctx); err != nil {
		err = fmt.Errorf("validate req: %w", err)
		return
	}

	credential, err := h.db.GetCredentialByID(ctx, credentialID)
	if err != nil {
		err = fmt.Errorf("get credential %s: %w", credentialID, err)
		return
	}

	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, credential.OwnerRef, permission.CredentialWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	if credential.Type == model.CredentialTypeOAuth {
		// only name can be updated
		credential.Name = req.Name
		err = h.db.UpdateCredentialByID(ctx, &credential, "name")
		if err != nil {
			err = fmt.Errorf("updating credential name: %w", err)
			return
		}

		OK(c, nil)
		return
	}

	var (
		encryptedData []byte
		inputFields   map[string]string
	)

	if req.OfficialName == "" {
		inputFields, err = h.getRealCredentialInputFields(credential, req.InputFields)
		if err != nil {
			err = fmt.Errorf("get actual credential encrypted data: %w", err)
			return
		}
	} else {
		// find the official credential config
		officialCredential, exists := h.getOfficialCredential(req.OfficialName)
		if !exists {
			err = fmt.Errorf("unknown official credential %q", req.OfficialName)
			return
		}

		inputFields = officialCredential.GetMergedData(req.InputFields)
	}

	encryptedData, err = h.decryptor.Marshal(inputFields)
	if err != nil {
		err = fmt.Errorf("encode credential data: %w", err)
		return
	}

	if !skipTesting {
		err = h.testCredential(ctx, req.AdapterClass, req.Type, inputFields)
		if err != nil {
			_ = c.Error(err)
			err = errTestCredentialFail
			return
		}
	}

	credential.Data = encryptedData
	credential.Name = req.Name
	err = h.db.UpdateCredentialByID(ctx, &credential, "name", "data")
	if err != nil {
		return
	}

	OK(c, nil)
}

// the mask field value of reqInputFields contains '***' if user not change it.
func (h *APIHandler) getRealCredentialInputFields(credential model.Credential, reqInputFields map[string]string) (inputFields map[string]string, err error) {
	err = h.decryptor.Unmarshal(credential.Data, &inputFields)
	if err != nil {
		err = fmt.Errorf("unmarshal credential data %s: %w", credential.ID, err)
		return
	}
	var credentialForm *adapter.CredentialForm
	credentialForm, err = getCredentialForm(credential.AdapterClass, string(credential.Type))
	if err != nil {
		err = fmt.Errorf("get credential form: %w", err)
		return
	}
	inputFields, _ = credentialForm.Masker.MergeChangedValue(inputFields, reqInputFields)
	return
}

// DeleteCredential
// @summary delete credential by id
// @produce json
// @param   id  path string true "credential id"
// @success 200 {object} apiserver.R
// @failure 400 {object} apiserver.R
// @router /api/v1/credentials/{id} [delete]
func (h *APIHandler) DeleteCredential(c *gin.Context) {
	var (
		ctx          = c.Request.Context()
		credentialID = c.Param("id")
		err          error
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	credential, err := h.db.GetCredentialByID(ctx, credentialID)
	if err != nil {
		return
	}
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, credential.OwnerRef, permission.CredentialWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	workflowIDsWithStatus, err := h.db.GetWorkflowIDWithStatusByCredentialID(ctx, credentialID)
	if err != nil {
		err = fmt.Errorf("getting enabled workflowIDs: %w", err)
		return
	}

	defer func() {
		if err != nil {
			return
		}
		err = h.db.DeleteCredentialByID(ctx, credentialID)
		if err != nil {
			err = fmt.Errorf("deleting credential: %w", err)
			return
		}

		// update nodes.credentialID to empty.
		if len(workflowIDsWithStatus) > 0 {
			workflowIDs := make([]string, len(workflowIDsWithStatus))
			for i := range workflowIDsWithStatus {
				workflowIDs[i] = workflowIDsWithStatus[i].ID
			}
			err = h.db.UpdateNodeCredentialEmptyByWorkflowIDsCredentialID(ctx, workflowIDs, credentialID)
			if err != nil {
				err = fmt.Errorf("updating node credential empty: %w", err)
				return
			}
		}

		OK(c, nil)
	}()

	// It's impossible to keep everything in a transaction, because disabling workflow depends on external io.
	// so just keep disabling each workflow in separate transaction.
	// must disable workflows first, then delete credential.
	for _, w := range workflowIDsWithStatus {
		if w.Status == model.WorkflowStatusEnabled {
			err = h.disableWorkflow(ctx, w.ID)
			if err != nil {
				err = fmt.Errorf("disabling workflow: %w", err)
				return
			}
		}
	}
}

// CredentialOAuth2Callback
// DEPRECATED: use /api/v1/credentials/oauth/* instead
func (h *APIHandler) CredentialOAuth2Callback(c *gin.Context) {
	var (
		ctx      = c.Request.Context()
		stateKey = c.Query("state")
		code     = c.Query("code")
		err      error
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if stateKey == "" || code == "" {
		err = errBizInvalidRequestPayload
		return
	}
	h.logger.For(ctx).Debug("oauth2 callback", log.String("stateKey", stateKey), log.String("code", code))

	stateStore := oauth2.NewDBStateStore(h.db)
	state, err := stateStore.GetStateByKey(ctx, stateKey)
	if err != nil {
		h.logger.For(ctx).Warn("state not found", log.String("stateKey", stateKey))
		err = errBizInvalidRequestPayload
		return
	}

	exchangeCode := func(oauth2Config xoauth2.Config) (token *xoauth2.Token) {
		token, err = oauth2Config.Exchange(ctx, code)
		if err != nil {
			h.logger.For(ctx).Error("exchange code error", log.ErrField(err))
			err = errBizInvalidRequestPayload
			return nil
		}
		return token
	}

	_, credentialData, err := h.getOAuth2CredentialData(c, state.CredentialID)
	if err != nil {
		return
	}

	// exchange code from oauth2 server
	oauth2Config := credentialData.GetOAuth2Config()
	token := exchangeCode(oauth2Config)
	if token == nil {
		return
	}

	// Handle slack access token including user token and bot token
	if strings.HasPrefix(oauth2Config.Endpoint.AuthURL, "https://slack.com/") {
		h.handleSlackCredential(c, state, token, credentialData)
		return
	}

	// salesforce return access token with zero expiry, it's default expiration is 2 hours
	if token.Expiry.IsZero() {
		token.Expiry = time.Now().Add(2 * time.Hour)
	}

	metaData, err := auth.GetTokenMetaData(token, credentialData)
	if err != nil {
		err = fmt.Errorf("getting token meta data: %w", err)
		return
	}

	tokenData, err := h.encodeTokenMeta(metaData, token)
	if err != nil {
		err = fmt.Errorf("encoding token meta: %w", err)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = h.db.UpdateCredentialTokenAndConfirmStatusByID(ctx, state.CredentialID, tokenData)
		if err != nil {
			err = fmt.Errorf("updating credential token: %w", err)
			return
		}

		state.Status = model.OAuth2StatusCompleted
		err = h.db.UpdateOAuth2StateByID(ctx, &state.OAuth2State, "status")
		if err != nil {
			err = fmt.Errorf("updating OAuth2State status: %w", err)
			return
		}
		return
	})
	if err != nil {
		h.logger.For(ctx).Error("updating database when oauth2 callback", log.ErrField(err), log.String("credentialID", state.CredentialID))
		return
	}

	if state.RedirectURL != "" {
		c.Redirect(http.StatusFound, state.RedirectURL)
		return
	}

	OK(c, nil)
}

// slack baseToken contains bot-token and user-token.
//
//	{
//	   "access_token": "xoxb-17653672481-19874698323-pdFZKVeTuE8sk7oOcBrzbqgy",
//	   "refresh_token": "xoxb-17653672481-19874698323-pdFZKVeTuE8sk7oOcBrzbqgy",
//	   "token_type": "bot",
//	   "authed_user": {
//	       "id": "U1234",
//	       "scope": "chat:write",
//	       "access_token": "xoxp-1234",
//	       "token_type": "user"
//	   }
//	}
func (h *APIHandler) handleSlackCredential(c *gin.Context, state *oauth2.State, baseToken *xoauth2.Token, credentialData *model.CredentialData) {
	var (
		ctx = c.Request.Context()
		err error
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	userToken, err := h.extractUserToken(baseToken)
	if err != nil {
		err = fmt.Errorf("extract user token: %w", err)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	botTokenID, err := h.extractAppIDAndTeamID(baseToken)
	if err != nil {
		err = fmt.Errorf("extract bot token id: %w", err)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	// read all metadata from base token.
	metaData, err := auth.GetTokenMetaData(baseToken, credentialData)
	if err != nil {
		err = fmt.Errorf("getting token meta data: %w", err)
		return
	}

	botTokenData, err := h.encodeTokenMeta(metaData, baseToken)
	if err != nil {
		err = fmt.Errorf("encode bot token data: %w", err)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	userTokenData, err := h.encodeTokenMeta(metaData, userToken)
	if err != nil {
		err = fmt.Errorf("encode user token data: %w", err)
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		userCredential, err := h.db.GetCredentialByID(ctx, state.CredentialID)
		if err != nil {
			err = fmt.Errorf("geting slack user credential: %w", err)
			return
		}
		botCredential := userCredential.Clone()
		botCredential.ID = botTokenID
		botCredential.Name = "slack_bot_token"
		botCredential.Status = model.CredentialStatusAvailable
		botCredential.Token = botTokenData
		botCredential.OwnerID = model.OwnerIDShare
		botCredential.OwnerType = model.OwnerTypeShare

		err = h.db.UpsertCredential(ctx, botCredential)
		if err != nil {
			err = fmt.Errorf("upserting credential: %w", err)
			return
		}
		err = h.db.UpdateCredentialTokenAndStatusWithParentID(ctx, state.CredentialID, botTokenID, userTokenData)
		if err != nil {
			err = fmt.Errorf("updating credential token: %w", err)
			return
		}

		state.Status = model.OAuth2StatusCompleted
		err = h.db.UpdateOAuth2StateByID(ctx, &state.OAuth2State, "status")
		if err != nil {
			err = fmt.Errorf("updating OAuth2State status: %w", err)
			return
		}
		return
	})

	if err != nil {
		h.logger.For(ctx).Error("updating database when oauth2 callback", log.ErrField(err), log.String("credentialID", state.CredentialID))
		return
	}

	if state.RedirectURL != "" {
		c.Redirect(http.StatusFound, state.RedirectURL)
		return
	}

	OK(c, nil)
}

// extractUserToken slack returns both of user-token and bot-token, and the user-token embedded in bot-token,
// so extract user-token from bot-token.
//
//	{
//	   "ok": true,
//	   "access_token": "xoxb-17653672481-19874698323-pdFZKVeTuE8sk7oOcBrzbqgy",
//	   "token_type": "bot",
//	   "scope": "commands,incoming-webhook",
//	   "bot_user_id": "U0KRQLJ9H",
//	   "app_id": "A0KRD7HC3",
//	   "team": {
//	       "name": "Slack Softball Team",
//	       "id": "T9TK3CUKW"
//	   },
//	   "enterprise": {
//	       "name": "slack-sports",
//	       "id": "E12345678"
//	   },
//	   "authed_user": {
//	       "id": "U1234",
//	       "scope": "chat:write",
//	       "access_token": "xoxp-1234",
//	       "token_type": "user"
//	   }
//	}
//
// ref doc: https://api.slack.com/authentication/oauth-v2#exchanging
func (h *APIHandler) extractUserToken(token *xoauth2.Token) (*xoauth2.Token, error) {
	userAuth := token.Extra("authed_user")
	userToken := &xoauth2.Token{}
	if val, ok := userAuth.(map[string]interface{}); ok {
		userToken.AccessToken = val["access_token"].(string)
		userToken.RefreshToken = val["refresh_token"].(string)
		userToken.TokenType = val["token_type"].(string)
		expires := val["expires_in"].(float64)
		if expires != 0 {
			userToken.Expiry = time.Now().Add(time.Duration(expires) * time.Second)
		}

		return userToken, nil
	}
	return nil, errors.New("get authed user error")
}

// extractAppIDAndTeamID slack bot token is global unique in ultrafox,
// botTokenID is composed of "slack", slack app id,team id and "bot"
func (h *APIHandler) extractAppIDAndTeamID(token *xoauth2.Token) (string, error) {
	appID := token.Extra("app_id").(string)
	team := token.Extra("team")
	var botTokenID string
	if val, ok := team.(map[string]interface{}); ok {
		teamID := val["id"].(string)
		botTokenID = "slack_bot:" + appID + "_" + teamID
		return botTokenID, nil
	}
	return "", errors.New("get team id error")
}

// encodeUserTokenMeta extracted user-token does not contain metaData,
// so use bot-token`s metaData
func (h *APIHandler) encodeUserTokenMeta(credentialData *model.CredentialData, userToken, botToken *xoauth2.Token) (tokenData []byte, err error) {
	metaData, err := auth.GetTokenMetaData(botToken, credentialData)
	if err != nil {
		err = fmt.Errorf("getting token meta data: %w", err)
		return
	}

	encodedToken := model.OAuth2Token{
		Token:    *userToken,
		MetaData: metaData,
	}

	return h.decryptor.Marshal(encodedToken)
}

func (h *APIHandler) encodeTokenMeta(metaData []byte, token *xoauth2.Token) (tokenData []byte, err error) {
	encodedToken := model.OAuth2Token{
		Token:    *token,
		MetaData: metaData,
	}

	return h.decryptor.Marshal(encodedToken)
}

func (h *APIHandler) getOAuth2CredentialData(c *gin.Context, credentialID string) (credential *model.Credential, credentialData *model.CredentialData, err error) {
	ctx := c.Request.Context()
	oauth2Credential, err := h.db.GetCredentialByID(ctx, credentialID)
	if err != nil {
		h.logger.For(ctx).Error("get credential error", log.String("credentialID", credentialID), log.ErrField(err))
		err = errBizReadDatabase
		return
	}
	if !oauth2Credential.Type.IsOAuth2() {
		err = errBizInvalidRequestPayload
		return
	}
	rawData, err := h.decryptor.Decrypt(oauth2Credential.Data)
	if err != nil {
		h.logger.For(ctx).Error("decode credential data error", log.ErrField(err))
		err = errInternal
		return
	}
	credentialData, err = service.ComposeCredentialData(oauth2Credential.AdapterClass, string(oauth2Credential.Type), rawData)
	if err != nil {
		h.logger.For(ctx).Error("compose credential data error", log.ErrField(err), log.String("credentialID", credentialID), log.ByteString("rawData", rawData))
		err = errInternal
		return
	}
	return &oauth2Credential, credentialData, nil
}

// RequestAuthURL
//
// DEPRECATED: use /api/v1/credentials/oauth/* instead
// @summary request official oauth2 authorization URL.
// @produce json
// @param   body body payload.RequestAuthURLReq true "body"
// @Success 200 {object} apiserver.R{data=response.RequestAuthURLResponse}
// @failure 400 {object} apiserver.R
// @router /api/v1/credentials/oauth2/authUrl [post]
func (h *APIHandler) RequestAuthURL(c *gin.Context) {
	var (
		ctx     = c.Request.Context()
		payload payload.RequestAuthURLReq
		err     error
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	err = c.ShouldBindJSON(&payload)
	if err != nil {
		err = errBizInvalidRequestPayload
		return
	}

	if err = payload.Validate(); err != nil {
		err = errBizInvalidRequestPayload
		return
	}

	credential, credentialData, err := h.getOAuth2CredentialData(c, payload.CredentialID)
	if err != nil {
		return
	}

	currentUserID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, currentUserID, credential.OwnerRef, permission.CredentialWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	var oauth2Config xoauth2.Config
	if payload.ForceRefresh && credential.OfficialName != "" {
		var exists bool
		officialCredential, exists := h.getOfficialCredential(credential.OfficialName)
		if !exists {
			err = errBizInvalidRequestPayload
			return
		}
		var rawData []byte
		rawData, err = json.Marshal(officialCredential.GetMergedData(nil))
		if err != nil {
			err = fmt.Errorf("marshaling official credential: %w", err)
			return
		}
		credentialData, err = service.ComposeCredentialData(officialCredential.Adapter, string(officialCredential.Type), rawData)
		if err != nil {
			err = fmt.Errorf("compose credential data: %w", err)
			return
		}
		// update the latest official oauth2 config to the credential data.
		credential.Data, err = h.decryptor.Marshal(credentialData)
		if err != nil {
			h.logger.For(ctx).Error("encrypt oauth2 config failed", log.ErrField(err), log.String("credentialID", credential.ID))
			return
		}
		err = h.db.UpdateCredentialByID(ctx, credential, "data")
		if err != nil {
			h.logger.For(ctx).Error("update credential failed", log.ErrField(err), log.String("credentialID", credential.ID))
			return
		}
		oauth2Config = credentialData.GetOAuth2Config()
	} else {
		oauth2Config = credentialData.GetOAuth2Config()
	}

	stateStore := oauth2.NewDBStateStore(h.db)
	stateKey, err := stateStore.AddState(ctx, oauth2.NewState(payload.CredentialID, payload.RedirectURL))
	if err != nil {
		h.logger.For(ctx).Error("add oauth2 state failed", log.ErrField(err))
		return
	}
	var authURL string
	if strings.HasPrefix(oauth2Config.Endpoint.AuthURL, "https://slack.com") {
		// slack get both of bot-token and user-token
		authURL = AuthCodeURL(&oauth2Config, stateKey)
	} else {
		authURL = oauth2Config.AuthCodeURL(stateKey, xoauth2.AccessTypeOffline)
	}

	OK(c, response.RequestAuthURLResponse{
		StateID: stateKey,
		AuthURL: authURL,
	})
}

func AuthCodeURL(c *xoauth2.Config, state string) string {
	var buf bytes.Buffer
	buf.WriteString(c.Endpoint.AuthURL)
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {c.ClientID},
		"access_type":   {"offline"},
	}
	if c.RedirectURL != "" {
		v.Set("redirect_uri", c.RedirectURL)
	}
	if len(c.Scopes) > 0 {
		v.Set("scope", strings.Join(c.Scopes, " "))
		v.Set("user_scope", strings.Join(c.Scopes, " "))
	}
	if state != "" {
		// TODO(nathan): Docs say never to omit state; don't allow empty.
		v.Set("state", state)
	}

	if strings.Contains(c.Endpoint.AuthURL, "?") {
		buf.WriteByte('&')
	} else {
		buf.WriteByte('?')
	}
	buf.WriteString(v.Encode())
	return buf.String()
}

func (h *APIHandler) getOfficialCredential(name string) (*model.OfficialCredential, bool) {
	for _, credential := range h.officialCredentials {
		if credential.Name == name {
			return credential, true
		}
	}
	return nil, false
}

// ListAssociatedWorkflows
// @summary get workflows are associated with given credential.
// @produce json
// @param   id  path string true "credential id"
// @Success 200 {object} apiserver.R{data=response.ListAssociatedWorkflowsResp}
// @failure 400 {object} apiserver.R
// @router /api/v1/credentials/{id}/associatedWorkflows [get]
func (h *APIHandler) ListAssociatedWorkflows(c *gin.Context) {
	var (
		ctx          = c.Request.Context()
		credentialID = c.Param("id")
		err          error
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	credential, err := h.db.GetCredentialByID(ctx, credentialID)
	if err != nil {
		err = fmt.Errorf("getting credential: %w", err)
		return
	}
	err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, credential.OwnerRef, permission.CredentialRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	workflows, err := h.db.GetEnabledWorkflowsByCredentialID(ctx, credentialID)
	if err != nil {
		err = fmt.Errorf("getting enabled workflowIDs: %w", err)
		return
	}

	OK(c, response.ListAssociatedWorkflowsResp{
		Workflows: workflows,
	})
}
