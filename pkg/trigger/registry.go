package trigger

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"
)

type Registry struct {
	logger               log.Logger
	serverHost           *serverhost.ServerHost
	nodeWebhookProviders map[string]*webhookProvider
	cipher               crypto.CryptoCipher
	dkronClient          *DkronClient

	// only use for OAuth token renewal
	dbNotInTx            *model.DB
	cache                *cache.Cache
	passportVendorLookup map[model.PassportVendorName]model.PassportVendor
}

type RegistryOpt struct {
	WebhookProviders     map[string]TriggerProvider
	Cipher               crypto.CryptoCipher
	ServerHost           *serverhost.ServerHost
	DkronConfig          DkronConfig
	DB                   *model.DB
	Cache                *cache.Cache
	PassportVendorLookup map[model.PassportVendorName]model.PassportVendor
}

// NewRegistry create a new trigger registry
func NewRegistry(opt RegistryOpt) (registry *Registry, err error) {
	dkronClient, err := NewDkronClient(opt.DkronConfig)
	if err != nil {
		err = fmt.Errorf("initializing Dkron client: %w", err)
		return
	}

	registry = &Registry{
		logger:               log.Clone(log.Namespace("trigger/registry")),
		serverHost:           opt.ServerHost,
		nodeWebhookProviders: buildInternalWebhookProvider(opt.WebhookProviders),
		cipher:               opt.Cipher,
		dkronClient:          dkronClient,
		dbNotInTx:            opt.DB,
		cache:                opt.Cache,
		passportVendorLookup: opt.PassportVendorLookup,
	}
	return
}

// EnableTrigger enable the specified trigger, external resource will be created if this kind of trigger needs (like gitlab),
// And update trigger.Data to store the external resource information.
//
// The param tx must be a database transaction, typically provided by model.DB.RunInTx
func (r *Registry) EnableTrigger(ctx context.Context, tx model.Operator, trigger *model.TriggerWithNode) (err error) {
	if trigger.Node == nil {
		return fmt.Errorf("trigger %s related node %s data must be provided", trigger.ID, trigger.NodeID)
	}

	switch trigger.Type {
	case model.TriggerTypeWebhook:
		return r.createWebhookTrigger(ctx, tx, *trigger)
	case model.TriggerTypeCron:
		return r.createCronTrigger(ctx, *trigger)
	case model.TriggerTypePoll:
		return r.createPollTrigger(ctx, tx, *trigger)
	default:
		return fmt.Errorf("unpected trigger type %q", trigger.Type)
	}
}

// DisableTrigger disable the specified trigger, external resource will be deleted if this kind of trigger needs (like gitlab),
// And update trigger.Data to remove the external resource information.
//
// The param tx must be a database transaction, typically provided by model.DB.RunInTx
func (r *Registry) DisableTrigger(ctx context.Context, tx model.Operator, trigger model.TriggerWithNode) (err error) {
	if trigger.Node == nil {
		return fmt.Errorf("trigger %s related node %s data must be provided", trigger.ID, trigger.NodeID)
	}

	if trigger.Node.Data.MetaData.EnableTriggerAtFirst {
		return nil
	}

	switch trigger.Type {
	case model.TriggerTypeWebhook:
		return r.deleteWebhookTrigger(ctx, tx, trigger)
	case model.TriggerTypeCron:
		return r.deleteCronTrigger(ctx, tx, trigger.Trigger)
	case model.TriggerTypePoll:
		return r.deletePollTrigger(ctx, tx, trigger)
	default:
		return fmt.Errorf("unpected trigger type %q", trigger.Type)
	}
}

func (r *Registry) getWebhookProvider(nodeClass string) (*webhookProvider, error) {
	provider, ok := r.nodeWebhookProviders[nodeClass]
	if !ok {
		return nil, fmt.Errorf("webhook provider not found for node %q", nodeClass)
	}
	return provider, nil
}

func (r *Registry) createWebhookTrigger(ctx context.Context, tx model.Operator, trigger model.TriggerWithNode) (err error) {
	node := *trigger.Node
	nodeClass := node.Class
	provider, err := r.getWebhookProvider(nodeClass)
	if err != nil {
		err = fmt.Errorf("getting webhook provider: %w", err)
		return
	}

	sign, err := r.getAuthorizer(ctx, tx, node)
	if err != nil {
		err = fmt.Errorf("getting authorizer: %w", err)
		return
	}

	// TODO(nathan): Here is a general method.
	// In the future, special processing may be done for some special adapters(like salesforce), which may require better abstraction
	var isSalesforce bool
	if strings.Contains(nodeClass, "salesforce") {
		isSalesforce = true
	}

	configObject := provider.GetConfigObject()
	if configObject != nil {
		var decoder *mapstructure.Decoder
		decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Squash: true,
			Result: configObject,
		})
		if err != nil {
			err = fmt.Errorf("initializing decoder: %w", err)
			return
		}

		err = decoder.Decode(node.Data.InputFields)
		if err != nil {
			err = fmt.Errorf("bind node input fields to trigger config object: %w", err)
			return
		}
	}

	webhookContext := newWebhookContext(webhookContextOpt{
		Ctx:                  ctx,
		ConfigObject:         configObject,
		Trigger:              trigger,
		AuthSignature:        sign,
		ServerHost:           r.serverHost,
		IsSalesforce:         isSalesforce,
		PassportVendorLookup: r.passportVendorLookup,
	})

	opt := oteltrace.WithAttributes(attribute.String("adapterClass", trigger.AdapterClass))
	_, createSpan := otel.Tracer("ultrafox").Start(ctx, "webhook.create", opt)
	defer createSpan.End()

	updateFields, err := provider.Create(webhookContext)
	if errors.Is(err, ErrTokenUnauthorized) {
		// print the error and carry on
		r.logger.Warn("credential is not valid when creating the webhook", zap.String("triggerId", trigger.ID), zap.Error(err))
		err = nil
	} else if err != nil {
		createSpan.SetStatus(codes.Error, err.Error())
		err = fmt.Errorf("creating webhook failed: %w", err)
		return
	}

	triggerData := updateFields
	// update the trigger with the new rawData
	err = tx.UpdateTriggerDataAndQueryIDByID(ctx, trigger.ID, triggerData, webhookContext.trigger.QueryID)
	if err != nil {
		err = fmt.Errorf("update trigger data and queryID: %w", err)
		return
	}
	return
}

func (r *Registry) deleteWebhookTrigger(ctx context.Context, tx model.Operator, trigger model.TriggerWithNode) (err error) {
	node := *trigger.Node
	nodeClass := node.Class
	provider, err := r.getWebhookProvider(nodeClass)
	if err != nil {
		err = fmt.Errorf("getting webhook provider: %w", err)
		return
	}

	sign, err := r.getAuthorizer(ctx, tx, node)
	if err != nil {
		err = fmt.Errorf("getting authorizer: %w", err)
		return
	}

	configObject := provider.GetConfigObject()
	if configObject != nil {
		var decoder *mapstructure.Decoder
		decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Squash: true,
			Result: configObject,
		})
		if err != nil {
			err = fmt.Errorf("initializing decoder: %w", err)
			return
		}

		err = decoder.Decode(node.Data.InputFields)
		if err != nil {
			err = fmt.Errorf("bind node input fields to trigger config object: %w", err)
			return
		}
	}

	opt := oteltrace.WithAttributes(attribute.String("adapterClass", trigger.AdapterClass))
	_, deleteSpan := otel.Tracer("ultrafox").Start(ctx, "webhook.delete", opt)
	defer deleteSpan.End()

	webhookContext := newWebhookContext(webhookContextOpt{
		Ctx:                  ctx,
		ConfigObject:         configObject,
		Trigger:              trigger,
		AuthSignature:        sign,
		ServerHost:           r.serverHost,
		PassportVendorLookup: r.passportVendorLookup,
	})

	err = provider.Delete(webhookContext)
	if errors.Is(err, ErrTokenUnauthorized) {
		// print the error and carry on
		r.logger.Warn("credential is not valid when removing the webhook", zap.String("triggerId", trigger.ID), zap.Error(err))
		err = nil
	} else if err != nil {
		deleteSpan.SetStatus(codes.Error, err.Error())
		err = fmt.Errorf("delete webhook failed: %w", err)
		return
	}
	deleteSpan.SetStatus(codes.Ok, "")

	err = tx.UpdateTriggerDataByID(ctx, trigger.ID, map[string]any{})
	if err != nil {
		err = fmt.Errorf("update trigger data: %w", err)
		return
	}

	return
}

// cronTriggerConfig ensemble struct of ultrafox/schedule inputs
type cronTriggerConfig struct {
	// timezone for cron expression
	Timezone string `json:"timezone"`
	// Dkron Expression
	Expression string `json:"expr"`

	// 10:35:00
	Time string `json:"time"`
	// 0-59
	Minute int `json:"minute"`
	// 1-31
	DaysOfMonth []int `json:"daysOfMonth"`
	// 0-6 as SUN-SAT
	WeekDays []int `json:"weekDays"`
}

func (r *Registry) createCronTrigger(ctx context.Context, trigger model.TriggerWithNode) (err error) {
	var input cronTriggerConfig
	err = utils.ConvertMapToStruct(trigger.Node.Data.InputFields, &input)
	if err != nil {
		err = fmt.Errorf("binding cron trigger config: %w", err)
		return
	}

	if input.Timezone == "" {
		err = errors.New("timezone is required")
		return
	}

	var (
		cronExpr             string
		hour, minute, second int
	)
	if input.Time != "" {
		hour, minute, second, err = ParseTimeInDay(input.Time)
		if err != nil {
			err = fmt.Errorf("parsing time in day: %w", err)
			return
		}
	}

	// Dkron expression: https://dkron.io/docs/v1/usage/cron-spec/
	//
	// Field name   | Mandatory? | Allowed values  | Allowed special characters
	// ----------   | ---------- | --------------  | --------------------------
	// Seconds      | Yes        | 0-59            | * / , -
	// Minutes      | Yes        | 0-59            | * / , -
	// Hours        | Yes        | 0-23            | * / , -
	// Day of month | Yes        | 1-31            | * / , - ?
	// Month        | Yes        | 1-12 or JAN-DEC | * / , -
	// Day of week  | Yes        | 0-6 or SUN-SAT  | * / , - ?
	switch trigger.Node.Class {
	case "ultrafox/schedule#everyDay":
		cronExpr = fmt.Sprintf("%d %d %d * * *", second, minute, hour)
	case "ultrafox/schedule#everyHour":
		cronExpr = fmt.Sprintf("0 %d * * * *", input.Minute)
	case "ultrafox/schedule#everyMonth":
		if len(input.DaysOfMonth) == 0 {
			err = errors.New("days of month is required for everyMonth cron")
			return
		}

		sort.Ints(input.DaysOfMonth)
		cronExpr = fmt.Sprintf("%d %d %d %s * *", second, minute, hour, joinInts(input.DaysOfMonth, ","))
	case "ultrafox/schedule#everyWeek":
		if len(input.WeekDays) == 0 {
			err = errors.New("days of week is required for everyMonth cron")
			return
		}

		sort.Ints(input.WeekDays)
		cronExpr = fmt.Sprintf("%d %d %d * * %s", second, minute, hour, joinInts(input.WeekDays, ","))
	case "ultrafox/schedule#cron":
		cronExpr = input.Expression
	default:
		err = fmt.Errorf("unexpected cron trigger class %q", trigger.Node.Class)
		return
	}

	_, err = r.dkronClient.UpsertJob(ctx, trigger.ID, cronExpr, input.Timezone, trigger.Node.Data.InputFields)
	if err != nil {
		err = fmt.Errorf("upserting cron job: %w", err)
		return
	}

	return
}

func (r *Registry) deleteCronTrigger(ctx context.Context, db model.Operator, trigger model.Trigger) error {
	err := r.dkronClient.DeleteJob(ctx, trigger.ID)
	if err != nil {
		return fmt.Errorf("deleting cron trigger %q: %w", trigger.ID, err)
	}

	return nil
}

// TODO(sword): read this configuration from adapter config file.
var noNeedCredentialsAdapters = set.FromSlice([]string{
	"ultrafox/debug",
	"ultrafox/webhook",
})

func (r *Registry) getAuthorizer(ctx context.Context, tx model.Operator, node model.Node) (sign auth.Authorizer, err error) {
	if node.CredentialID == "" {
		if noNeedCredentialsAdapters.Has(node.Data.MetaData.AdapterClass) {
			// Those special adapters does not need credentials
			return
		}

		err = fmt.Errorf("credential id of node %q is empty", node.ID)
		return
	}

	var credential model.Credential
	credential, err = tx.GetAvailableCredentialByID(ctx, node.CredentialID)
	if err != nil {
		err = fmt.Errorf("querying auth connection with credential: %w", err)
		return
	}

	sign, err = auth.NewAuthorizer(r.cipher, &credential,
		auth.WithUpdateCredentialTokenFunc(r.dbNotInTx.UpdateCredentialTokenAndConfirmStatusByID),
		auth.WithOAuthCredentialUpdater(auth.OAuthCredentialUpdater{DB: r.dbNotInTx.Operator, Cache: r.cache}),
	)
	if err != nil {
		err = fmt.Errorf("auth.NewAuthorizer: %w", err)
		return
	}

	return
}

func (r *Registry) deletePollTrigger(ctx context.Context, tx model.Operator, trigger model.TriggerWithNode) (err error) {
	err = tx.UpdateTriggerDataByID(ctx, trigger.ID, map[string]any{})
	if err != nil {
		err = fmt.Errorf("update trigger data: %w", err)
		return
	}

	return
}

func (r *Registry) createPollTrigger(ctx context.Context, tx model.Operator, trigger model.TriggerWithNode) (err error) {
	provider, err := r.getWebhookProvider(trigger.Node.Class)
	if err != nil {
		err = fmt.Errorf("getting provider: %w", err)
		return
	}

	configObject := provider.GetConfigObject()
	if configObject != nil {
		var decoder *mapstructure.Decoder
		decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Squash: true,
			Result: configObject,
		})
		if err != nil {
			err = fmt.Errorf("initializing decoder: %w", err)
			return
		}

		err = decoder.Decode(trigger.Node.Data.InputFields)
		if err != nil {
			err = fmt.Errorf("binding node input fields to trigger config object: %w", err)
			return
		}
	}

	// validate user API token before enable trigger
	authorizer, err := r.getAuthorizer(ctx, tx, *trigger.Node)
	if err != nil {
		err = fmt.Errorf("getting authorizer: %w", err)
		return
	}

	webhookContext := newWebhookContext(webhookContextOpt{
		Ctx:                  ctx,
		ConfigObject:         configObject,
		AuthSignature:        authorizer,
		ServerHost:           r.serverHost,
		Trigger:              trigger,
		IsSalesforce:         false,
		PassportVendorLookup: r.passportVendorLookup,
	})
	updateFields, err := provider.Create(webhookContext)
	if err != nil {
		err = fmt.Errorf("provider creating: %w", err)
		return
	}

	triggerData := updateFields
	// update the trigger with the new rawData
	err = tx.UpdateTriggerDataAndQueryIDByID(ctx, trigger.ID, triggerData, webhookContext.trigger.QueryID)
	if err != nil {
		err = fmt.Errorf("update trigger data and queryID: %w", err)
		return
	}
	return
}

// ParseTimeInDay parses time like 16:23:45
func ParseTimeInDay(timeInDay string) (hour, min, second int, err error) {
	if len(timeInDay) != len("00:00:00") {
		err = fmt.Errorf("unexpected length of input %q", timeInDay)
		return
	}
	for _, char := range timeInDay {
		// ASCII 48-58: https://www.rapidtables.com/code/text/ascii-table.html
		if char < '0' || char > ':' {
			err = fmt.Errorf("unexpected char %q in input", char)
			return
		}
	}

	n, err := fmt.Fscanf(strings.NewReader(timeInDay), "%02d:%02d:%02d", &hour, &min, &second)
	if err != nil {
		err = fmt.Errorf("scaning time in day: %w", err)
		return
	}
	if n != 3 {
		err = fmt.Errorf("expected scan 3 items, got %d", n)
		return
	}

	if hour < 0 || hour > 23 {
		err = fmt.Errorf("invalid hour %d", hour)
		return
	}
	if min < 0 || min > 59 {
		err = fmt.Errorf("invalid minute %d", min)
		return
	}
	if second < 0 || second > 59 {
		err = fmt.Errorf("invalid second %d", second)
		return
	}

	return
}

func joinInts(ints []int, sep string) string {
	if len(ints) == 0 {
		return ""
	}

	var buf strings.Builder

	buf.WriteString(strconv.Itoa(ints[0]))
	for _, i := range ints[1:] {
		buf.WriteString(sep)
		buf.WriteString(strconv.Itoa(i))
	}

	return buf.String()
}
