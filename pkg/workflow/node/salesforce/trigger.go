package salesforce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"

	"github.com/spf13/cast"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/common"
)

var (
	_ trigger.TriggerProvider = (*AccountCreatedTrigger)(nil)
	_ trigger.TriggerProvider = (*AccountUpdatedTrigger)(nil)
	_ trigger.TriggerProvider = (*OpportunityCreatedTrigger)(nil)
	_ trigger.TriggerProvider = (*OpportunityUpdatedTrigger)(nil)
	_ trigger.TriggerProvider = (*LicenseCreateTrigger)(nil)
)

var (
	className    = "ultrafox/salesforce"
	adapterClass = common.AdapterClass(className)
)

const (
	APIVERSION         = "55.0"
	APEXCLASSKEY       = "apexClassId"
	APEXTRIGGERKEY     = "apexTriggerId"
	APEXTRIGGERTESTKEY = "apexTriggerTestId"
	APEXCLASSNAME      = "apexTriggerTestName"
	ErrorDuplicate     = "DUPLICATE_VALUE"
)

var triggerEvents = []string{
	"before insert",
	"before update",
	"before delete",
	"after insert",
	"after update",
	"after delete",
}

type baseTrigger struct{}

func (t *baseTrigger) GetConfigObject() any {
	return &salesforceEventConfig{}
}

func (t *baseTrigger) FieldsDeleted() []string {
	return []string{
		APEXCLASSKEY,
		APEXTRIGGERKEY,
		APEXTRIGGERTESTKEY,
		APEXCLASSNAME,
	}
}

func (t *baseTrigger) CreateTrigger(c trigger.WebhookContext, config *salesforceEventConfig) (map[string]any, error) {
	client, err := newSalesforceClient(c)
	if err != nil {
		return nil, fmt.Errorf("new salesforce client: %w", err)
	}
	config.SessionId = client.accessToken
	return client.triggerCreate(config)
}

func (t *baseTrigger) Delete(c trigger.WebhookContext) error {
	client, err := newSalesforceClient(c)
	if err != nil {
		return fmt.Errorf("new salesforce client: %w", err)
	}
	return client.triggerDelete(c)
}

type LicenseCreateTrigger struct {
	baseTrigger
	Body []byte `json:"body"`
	ApexTriggerData
}

func (t *LicenseCreateTrigger) Run(c *workflow.NodeContext) (any, error) {
	data := map[string]any{}
	err := json.Unmarshal(t.Body, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return data, nil
}

func (t *LicenseCreateTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	config, err := newSalesforceApexConfig(c)
	if err != nil {
		return nil, fmt.Errorf("new salesforce apex config: %w", err)
	}
	config.SObject = "Release_License_Application__c"
	config.Events = []string{"after insert"}
	return t.CreateTrigger(c, config)
}

func (t *LicenseCreateTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("licenseCreatedTrigger"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(LicenseCreateTrigger)
		},
	}
}

func (t *LicenseCreateTrigger) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	return getSObjectCollection(ctx, "Release_License_Application__c")
}

type AccountCreatedTrigger struct {
	baseTrigger
	Body []byte `json:"body"`
	ApexTriggerData
}

func (t *AccountCreatedTrigger) Run(c *workflow.NodeContext) (any, error) {
	data := map[string]any{}
	err := json.Unmarshal(t.Body, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return data, nil
}

func (t *AccountCreatedTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	config, err := newSalesforceApexConfig(c)
	if err != nil {
		return nil, fmt.Errorf("new salesforce apex config: %w", err)
	}
	config.SObject = "account"
	config.Events = []string{"after insert"}
	return t.CreateTrigger(c, config)
}

type AccountUpdatedTrigger struct {
	baseTrigger
	Body []byte `json:"body"`
	ApexTriggerData
}

func (t *AccountUpdatedTrigger) Run(c *workflow.NodeContext) (any, error) {
	data := map[string]any{}
	err := json.Unmarshal(t.Body, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return data, nil
}

func (t *AccountUpdatedTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	config, err := newSalesforceApexConfig(c)
	if err != nil {
		return nil, fmt.Errorf("new salesforce apex config: %w", err)
	}
	config.SObject = "account"
	config.Events = []string{"after update"}
	return t.CreateTrigger(c, config)
}

type OpportunityCreatedTrigger struct {
	baseTrigger
	Body []byte `json:"body"`
	ApexTriggerData
}

func (t *OpportunityCreatedTrigger) Run(c *workflow.NodeContext) (any, error) {
	data := map[string]any{}
	err := json.Unmarshal(t.Body, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return data, nil
}

func (t *OpportunityCreatedTrigger) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	return getSObjectCollection(ctx, "opportunity")
}

func (t *OpportunityCreatedTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	config, err := newSalesforceApexConfig(c)
	if err != nil {
		return nil, fmt.Errorf("new salesforce apex config: %w", err)
	}
	config.SObject = "opportunity"
	config.Events = []string{"after insert"}
	return t.CreateTrigger(c, config)
}

type OpportunityUpdatedTrigger struct {
	baseTrigger
	Body []byte `json:"body"`
	ApexTriggerData
}

func (t *OpportunityUpdatedTrigger) Run(c *workflow.NodeContext) (any, error) {
	data := map[string]any{}
	err := json.Unmarshal(t.Body, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return data, nil
}

func (t *OpportunityUpdatedTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	config, err := newSalesforceApexConfig(c)
	if err != nil {
		return nil, fmt.Errorf("new salesforce apex config: %w", err)
	}
	config.SObject = "opportunity"
	config.Events = []string{"after update"}
	return t.CreateTrigger(c, config)
}

func (t *OpportunityUpdatedTrigger) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	return getSObjectCollection(ctx, "opportunity")
}

func (t *OpportunityUpdatedTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("opportunityUpdatedTrigger"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(OpportunityUpdatedTrigger)
		},
	}
}

func (t *OpportunityCreatedTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("opportunityCreatedTrigger"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(OpportunityCreatedTrigger)
		},
	}
}

func (t *AccountUpdatedTrigger) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	return getSObjectCollection(ctx, "account")
}

func (t *AccountUpdatedTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("accountUpdatedTrigger"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(AccountUpdatedTrigger)
		},
	}
}

func (t *AccountCreatedTrigger) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	return getSObjectCollection(ctx, "account")
}

func (t *AccountCreatedTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("accountCreatedTrigger"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(AccountCreatedTrigger)
		},
	}
}

type salesforceEventConfig struct {
	TriggerName string   `json:"triggerName,omitempty"`
	SObject     string   `json:"sobject"`
	Events      []string `json:"events"`
	SessionId   string   `json:"sessionId,omitempty"`
	CallbackUrl string   `json:"callbackUrl,omitempty"`
}

type ErrorRespData struct {
	Message   string   `json:"message,omitempty"`
	ErrorCode string   `json:"errorCode,omitempty"`
	Fields    []string `json:"fields,omitempty"`
}
type ErrorRespDataArray []ErrorRespData

type SuccessRespData struct {
	ID       string   `json:"id,omitempty"`
	Success  bool     `json:"success,omitempty"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Infos    []string `json:"infos,omitempty"`
}

type ApexTriggerData struct {
	TriggerName string   `json:"triggerName"`
	SObject     string   `json:"sObject"`
	Events      []string `json:"events"`
	CallbackUrl string   `json:"callbackUrl"`
}

type SFDCPotos struct {
	Picture   string `json:"picture"`
	Thumbnail string `json:"thumbnail"`
}

type SFDCURL struct {
	Enterprise   string `json:"enterprise"`
	Metadata     string `json:"metadata"`
	Partner      string `json:"partner"`
	Rest         string `json:"rest"`
	SObjects     string `json:"sobjects"`
	Search       string `json:"search"`
	Query        string `json:"query"`
	Recent       string `json:"recent"`
	ToolingSoap  string `json:"tooling_soap"`
	ToolingRest  string `json:"tooling_rest"`
	Profile      string `json:"profile"`
	Feeds        string `json:"feeds"`
	Groups       string `json:"groups"`
	Users        string `json:"users"`
	FeedItems    string `json:"feed_items"`
	FeedElements string `json:"feed_elements"`
	CustomDomain string `json:"custom_domain"`
}

type SFDCAddress struct {
	Country string `json:"country"`
}

type SFDCUserInfo struct {
	Sub               string      `json:"sub"`
	UserId            string      `json:"user_id"`
	OrganizationId    string      `json:"organization_id"`
	PreferredUsername string      `json:"preferred_username"`
	Nickname          string      `json:"nickname"`
	Name              string      `json:"name"`
	Email             string      `json:"email"`
	EmailVerified     bool        `json:"email_verified"`
	FamilyName        string      `json:"family_name"`
	Zoneinfo          string      `json:"zoneinfo"`
	Potos             SFDCPotos   `json:"photos"`
	Profile           string      `json:"profile"`
	Picture           string      `json:"picture"`
	Address           SFDCAddress `json:"address"`
	URLs              SFDCURL     `json:"urls"`
	Active            bool        `json:"active"`
	UserType          string      `json:"user_type"`
	Language          string      `json:"language"`
	Locale            string      `json:"locale"`
	UTCOffset         int64       `json:"utcOffset"`
	UpdatedAt         string      `json:"updated_at"`
	IsAppInstalled    bool        `json:"is_app_installed"`
}

func GenerateApexCode(config *salesforceEventConfig, name, tpl string) (string, error) {
	triggerTpl := template.Must(template.New(name).Funcs(map[string]any{"join": strings.Join}).Parse(tpl))
	var buffer bytes.Buffer
	err := triggerTpl.Execute(&buffer, config)
	if err != nil {
		return "", fmt.Errorf("generate apex code: %w", err)
	}
	return buffer.String(), nil
}

type salesforceServerMeta struct {
	Server string `json:"server"`
}

func (u *SFDCUserInfo) restURL() string {
	return strings.Replace(u.URLs.Rest, "{version}", APIVERSION, -1)
}

func (u *SFDCUserInfo) metaURL() string {
	return strings.Replace(u.URLs.Metadata, "{version}", APIVERSION, -1)
}

func (u *SFDCUserInfo) queryURL() string {
	return strings.Replace(u.URLs.Query, "{version}", APIVERSION, -1)
}

func (u *SFDCUserInfo) sobjectURL() string {
	return strings.Replace(u.URLs.SObjects, "{version}", APIVERSION, -1)
}

func newSalesforceClient(t trigger.BaseContext) (*Client, error) {
	accessToken, err := t.GetAuthorizer().GetAccessToken(t.Context())
	if err != nil {
		return nil, fmt.Errorf("get salesforce access token error: %w", err)
	}

	meta := &salesforceServerMeta{}
	err = t.GetAuthorizer().DecodeMeta(meta)
	if err != nil {
		return nil, fmt.Errorf("decode auth meta data: %w", err)
	}

	sfdcClient := NewClient(accessToken, meta.Server)
	return sfdcClient, nil
}

func toJSON(data map[string]interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(data)
	return buffer.Bytes(), err
}

func getString(params map[string]any, key string) string {
	value, ok := params[key]
	if !ok {
		return ""
	}
	return cast.ToString(value)
}

func newSalesforceApexConfig(t trigger.WebhookContext) (*salesforceEventConfig, error) {
	config := t.GetConfigObject().(*salesforceEventConfig)
	randomId, err := utils.NanoID()
	if err != nil {
		return nil, fmt.Errorf("generate random id: %w", err)
	}
	config.TriggerName = randomId
	config.CallbackUrl = t.GetWebhookURL()
	return config, nil
}

func getSObjectCollection(ctx trigger.BaseContext, sobject string) (result []trigger.SampleData, err error) {
	client, err := newSalesforceClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize salesforce client: %w", err)
	}
	resp, err := client.getSObjectCollection(sobject, nil, 10, -1)
	if err != nil {
		return nil, fmt.Errorf("get sobject collection error: %w", err)
	}
	total := 10
	if resp.TotalSize < 10 {
		total = resp.TotalSize
	}

	result = make([]trigger.SampleData, total)
	for i, record := range resp.Records {
		result[i] = record
	}
	return result, nil
}
