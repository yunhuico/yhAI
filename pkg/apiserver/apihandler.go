package apiserver

import (
	"errors"
	"fmt"
	"net/url"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
)

// R is the response envelope
type R = httpbase.R

// OK responds the client with standard JSON.
//
// Example:
// * OK(c, something)
// * OK(c, nil)
var OK = httpbase.OK

type APIHandler struct {
	db              *model.DB
	cache           *cache.Cache
	logger          log.Logger
	triggerRegistry *trigger.Registry
	decryptor       crypto.CryptoCipher
	enforcer        *permission.Enforcer
	// key: vendor's name
	passportVendorLookup map[model.PassportVendorName]model.PassportVendor
	serverHost           *serverhost.ServerHost
	cipher               crypto.CryptoCipher
	officialCredentials  model.OfficialCredentials
	mailSender           *smtp.Sender
	workProducer         *work.Producer
	betaConfig           BetaConfig
}

type BetaConfig struct {
	// URL to the invitation sign-up collection sheet
	InvitationSignUpSheetURL string `comment:"URL to the invitation sign-up collection sheet"`
	// field id of the sign-up sheet at vika.cn, e.g. fldNNWWMQfJKG
	InvitationSignUpSheetEmailFieldID string `comment:"field id of the sign-up sheet at vika.cn, e.g. fldNNWWMQfJKG"`
	// bearer token for the special API that approves user beta invitation
	APIBearerToken string `comment:"bearer token for the special API that approves user beta invitation"`
}

func (c BetaConfig) InvitationSignUpSheetURLWithEmail(email string) string {
	// keeps the old behavior before the config is updated
	if c.InvitationSignUpSheetEmailFieldID == "" || email == "" {
		return c.InvitationSignUpSheetURL
	}
	parsed, err := url.Parse(c.InvitationSignUpSheetURL)
	if err != nil {
		return c.InvitationSignUpSheetURL
	}
	if !parsed.IsAbs() {
		return c.InvitationSignUpSheetURL
	}

	q := parsed.Query()
	q.Add(c.InvitationSignUpSheetEmailFieldID, email)
	parsed.RawQuery = q.Encode()

	return parsed.String()
}

type APIHandlerOpt struct {
	DB                        *model.DB
	Cache                     *cache.Cache
	Cipher                    crypto.CryptoCipher
	TriggerRegistry           *trigger.Registry
	PassportVendors           model.PassportVendors
	ServerHost                *serverhost.ServerHost
	OfficialOAuth2Credentials model.OfficialCredentials
	mailSender                *smtp.Sender
	WorkProducer              *work.Producer
	BetaConfig                BetaConfig
}

func newAPIHandler(opt APIHandlerOpt) (handler *APIHandler, err error) {
	passportVendorLookup, err := opt.PassportVendors.MapByVendorName()
	if err != nil {
		err = fmt.Errorf("building passportVendorLookup: %w", err)
		return
	}
	if opt.BetaConfig.APIBearerToken == "" {
		err = errors.New("APIBearerToken is required")
		return
	}
	if opt.BetaConfig.InvitationSignUpSheetURL == "" {
		err = errors.New("InvitationSignUpSheetURL is required")
		return
	}

	handler = &APIHandler{
		db:                   opt.DB,
		cache:                opt.Cache,
		logger:               log.Clone(log.Namespace("workflow/apiHandler")),
		triggerRegistry:      opt.TriggerRegistry,
		decryptor:            opt.Cipher,
		enforcer:             permission.NewEnforcer(opt.DB),
		passportVendorLookup: passportVendorLookup,
		serverHost:           opt.ServerHost,
		cipher:               opt.Cipher,
		officialCredentials:  opt.OfficialOAuth2Credentials,
		mailSender:           opt.mailSender,
		workProducer:         opt.WorkProducer,
		betaConfig:           opt.BetaConfig,
	}

	return
}

func extractPageParameters(c *gin.Context) (limit, offset int, orderBy string, err error) {
	type pageQuery struct {
		Limit   int    `form:"limit"`
		Offset  int    `form:"offset"`
		OrderBy string `form:"orderBy"`
	}

	limit = 20
	offset = 0
	orderBy = "created_at"
	query := pageQuery{}
	err = c.ShouldBindQuery(&query)
	if err != nil {
		err = errBizInvalidRequestPayload
		return
	}
	if 0 < query.Limit && query.Limit <= 100 {
		limit = query.Limit
	}
	if 0 < query.Offset {
		offset = query.Offset
	}
	if query.OrderBy == "created_at" || query.OrderBy == "updated_at" {
		orderBy = query.OrderBy
	}

	return limit, offset, orderBy, nil
}
