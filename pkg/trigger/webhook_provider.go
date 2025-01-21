package trigger

import (
	"context"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

// ErrTokenUnauthorized is used to signal that the credential is not valid
// so that there's no point to retry the action.
var ErrTokenUnauthorized = errors.New("the credential/token is not valid")

var _ WebhookContext = (*webhookContext)(nil)

type (
	// BaseContext provides basic dependencies for trigger node.
	BaseContext interface {
		Context() context.Context
		GetConfigObject() any
		GetAuthorizer() auth.Authorizer
	}

	// WebhookContext is the context for create, delete the webhook resource
	WebhookContext interface {
		log.Logger

		BaseContext

		GetTriggerData() map[string]any
		GetWebhookURL() string
		SetTriggerQueryID(queryID string)
		GetPassportVendorLookup() map[model.PassportVendorName]model.PassportVendor
	}

	TriggerProvider interface {
		// GetConfigObject get the config object
		GetConfigObject() any

		// Create external webhook resource
		Create(c WebhookContext) (map[string]any, error)

		// Delete external webhook resource
		Delete(c WebhookContext) error
	}

	webhookContext struct {
		log.Logger

		TriggerDeps

		trigger    model.TriggerWithNode
		serverHost *serverhost.ServerHost
		// TODO(nathan): Here is a general method.
		// In the future, special processing may be done for some special adapters(like salesforce), which may require better abstraction
		IsSalesforce bool
	}

	webhookProvider struct {
		TriggerProvider
	}

	TriggerDeps struct {
		context              context.Context
		authorizer           auth.Authorizer
		configObject         any
		passportVendorLookup map[model.PassportVendorName]model.PassportVendor
	}
)

func (w *webhookContext) GetWebhookURL() string {
	if w.IsSalesforce {
		return fmt.Sprintf("%s/salesforce/hooks/%s", w.serverHost.Webhook(), w.trigger.ID)
	}
	return fmt.Sprintf("%s/hooks/%s", w.serverHost.Webhook(), w.trigger.ID)
}

func (w *webhookContext) GetTriggerData() map[string]any {
	return w.trigger.Data
}

func (w *webhookContext) SetTriggerQueryID(queryID string) {
	w.trigger.QueryID = queryID
}

type webhookContextOpt struct {
	Ctx           context.Context
	ConfigObject  any
	AuthSignature auth.Authorizer
	ServerHost    *serverhost.ServerHost
	Trigger       model.TriggerWithNode
	// TODO(nathan): Here is a general method.
	// In the future, special processing may be done for some special adapters(like salesforce), which may require better abstraction
	IsSalesforce         bool
	PassportVendorLookup map[model.PassportVendorName]model.PassportVendor
}

func newWebhookContext(opt webhookContextOpt) *webhookContext {
	return &webhookContext{
		Logger: log.Clone(log.Namespace("webhookContext")),
		TriggerDeps: TriggerDeps{
			context:              opt.Ctx,
			authorizer:           opt.AuthSignature,
			configObject:         opt.ConfigObject,
			passportVendorLookup: opt.PassportVendorLookup,
		},
		trigger:      opt.Trigger,
		serverHost:   opt.ServerHost,
		IsSalesforce: opt.IsSalesforce,
	}
}

func NewBaseContext(ctx context.Context, authorizer auth.Authorizer, configObj any, passportVendorLookup map[model.PassportVendorName]model.PassportVendor) *TriggerDeps {
	return &TriggerDeps{
		context:              ctx,
		authorizer:           authorizer,
		configObject:         configObj,
		passportVendorLookup: passportVendorLookup,
	}
}

func buildInternalWebhookProvider(providers map[string]TriggerProvider) map[string]*webhookProvider {
	result := make(map[string]*webhookProvider, len(providers))
	for nodeClass, provider := range providers {
		result[nodeClass] = &webhookProvider{
			TriggerProvider: provider,
		}
	}
	return result
}

func (w *TriggerDeps) Context() context.Context {
	return w.context
}

func (w *TriggerDeps) GetConfigObject() any {
	return w.configObject
}

func (w *TriggerDeps) GetAuthorizer() auth.Authorizer {
	return w.authorizer
}

func (w *TriggerDeps) GetPassportVendorLookup() map[model.PassportVendorName]model.PassportVendor {
	return w.passportVendorLookup
}
