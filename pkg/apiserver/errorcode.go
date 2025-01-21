package apiserver

import (
	"net/http"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
)

const (
	// codeUnauthorized stands for invalid token,
	// which is an umbrella error exposed to public
	codeUnauthorized = 600401
	// codeInvalidParameter stands for invalid request
	codeInvalidParameter = 600402
	// codeResourceNotFound for resource not found
	codeResourceNotFound = 600404
	// codeReadDatabaseError for database read error
	codeReadDatabaseError = 600406
	// codeInternalError for invalid error
	codeInternalError = 600410
	// codeRunNodeError occurs in the workflow editing process.
	codeRunNodeError = 600413
	// codeNoMoreSamplesToLoad cannot load more samples
	codeNoMoreSamplesToLoad = 600414
	codeInvalidConfirmation = 600415
	// codeNoPermission user has logged in, but does not have enough permissions to access the resources.
	codeNoPermission                = 600416
	codeTestCredentialFail          = 600417
	codeNodesShouldTestBeforeActive = 600418
)

const (
	msgKeyUnauthorized                = "unauthorized"
	msgKeyInvalidParameter            = "invalidParameter"
	msgKeyResourceNotFound            = "resourceNotFound"
	msgKeyInternalError               = "internalError"
	msgKeyRunNodeError                = "runNodeError"
	msgKeyUserAlreadyJoinedOrg        = "userAlreadyJoinedOrg"
	msgKeyNoMoreSamplesToLoad         = "noMoreSamplesToLoad"
	msgKeyInvalidConfirm              = "invalidConfirm"
	msgKeyNoPermission                = "permissionDenied"
	msgKeyTestCredentialFail          = "testCredentialFail"
	msgKeyNodesShouldTestBeforeActive = "nodesShouldTestBeforeActive"
)

var (
	// errUnauthorized user needs login
	errUnauthorized = httpbase.NewError(http.StatusUnauthorized, codeUnauthorized, "Unauthorized", msgKeyUnauthorized)
	// errBizInvalidWorkflowID invalid workflow id
	errBizInvalidWorkflowID          = httpbase.NewError(http.StatusBadRequest, codeInvalidParameter, "Invalid workflow id", msgKeyInvalidParameter)
	errBizInvalidWorkflowInstanceID  = httpbase.NewError(http.StatusBadRequest, codeInvalidParameter, "Invalid workflow instance id", msgKeyInvalidParameter)
	errBizInvalidRequestPayload      = httpbase.NewError(http.StatusBadRequest, codeInvalidParameter, "Invalid request payload", msgKeyInvalidParameter)
	errBizInvalidWorkflowShareLinkID = httpbase.NewError(http.StatusBadRequest, codeInvalidParameter, "Invalid workflow link id", msgKeyInvalidParameter)
	// errBizInvitationNotFound the org invitation id refers to nothing.
	errBizInvitationNotFound        = httpbase.NewError(http.StatusOK, codeResourceNotFound, "Invalid invitation", msgKeyResourceNotFound)
	errBizWorkflowShareLinkNotFound = httpbase.NewError(http.StatusOK, codeResourceNotFound, "Workflow link not found", msgKeyResourceNotFound)
	errBizWorkflowNotFound          = httpbase.NewError(http.StatusOK, codeResourceNotFound, "Workflow not found", msgKeyResourceNotFound)
	// errBizUserAlreadyOrgMember User is already an organization member
	errBizUserAlreadyOrgMember     = httpbase.NewError(http.StatusOK, codeInvalidParameter, "User is already an organization member", msgKeyUserAlreadyJoinedOrg)
	errBizReadDatabase             = httpbase.NewError(http.StatusBadRequest, codeReadDatabaseError, "Read database error", msgKeyInternalError)
	errInternal                    = httpbase.NewError(http.StatusOK, codeInternalError, "Internal error", msgKeyInternalError)
	errBizWorkflowCheckFail        = httpbase.NewError(http.StatusOK, codeInternalError, "Workflow check fail", msgKeyInternalError)
	errRunNodeFailed               = httpbase.NewError(http.StatusOK, codeRunNodeError, "run node failed, check log or retry", msgKeyRunNodeError)
	errNoPermissionError           = httpbase.NewError(http.StatusOK, codeNoPermission, "Permission denied", msgKeyNoPermission)
	errCreateNodeDenied            = httpbase.NewError(http.StatusOK, codeInternalError, "create node denied", msgKeyInternalError)
	errDeleteNodeDenied            = httpbase.NewError(http.StatusOK, codeInternalError, "delete node denied", msgKeyInternalError)
	errUpdateNodeDenied            = httpbase.NewError(http.StatusOK, codeInternalError, "update node denied", msgKeyInternalError)
	errDeleteWorkflowDenied        = httpbase.NewError(http.StatusOK, codeInternalError, "delete workflow failed when workflow is active", msgKeyInternalError)
	errNoMoreSamplesToLoad         = httpbase.NewError(http.StatusOK, codeNoMoreSamplesToLoad, "no more samples to load", msgKeyNoMoreSamplesToLoad)
	errInvalidConfirmation         = httpbase.NewError(http.StatusOK, codeInvalidConfirmation, "Confirm is missing or invalid, or the workflow is disabled", msgKeyInvalidConfirm)
	errTestCredentialFail          = httpbase.NewError(http.StatusOK, codeTestCredentialFail, "Test credential fail", msgKeyTestCredentialFail)
	errNodesShouldTestBeforeActive = httpbase.NewError(http.StatusOK, codeNodesShouldTestBeforeActive, "Activate workflow fail", msgKeyNodesShouldTestBeforeActive)
)
