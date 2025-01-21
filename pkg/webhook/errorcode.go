package webhook

import (
	"net/http"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
)

var msgKeyTriggerError = "webhookError"

var (
	errInvalidWebhookCall = httpbase.NewError(http.StatusNotAcceptable, http.StatusNotAcceptable, "invalid webhook call", msgKeyTriggerError)
	errInvalidWebhookID   = httpbase.NewError(http.StatusNotAcceptable, http.StatusNotAcceptable, "invalid webhook id", msgKeyTriggerError)
	errInvalidEvent       = httpbase.NewError(http.StatusBadRequest, http.StatusBadRequest, "invalid event call", msgKeyTriggerError)
	errUnauthorized       = httpbase.NewError(http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized event", msgKeyTriggerError)
	errInvalidSObject     = httpbase.NewError(http.StatusNotAcceptable, http.StatusNotAcceptable, "invalid salesforce object", msgKeyTriggerError)
	errInvalidTrigger     = httpbase.NewError(http.StatusNotAcceptable, http.StatusNotAcceptable, "invalid trigger", msgKeyTriggerError)
)
