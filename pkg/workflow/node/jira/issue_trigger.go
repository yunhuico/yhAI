package jira

import (
	"fmt"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/jira"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var (
	_ workflow.Node           = (*IssueTrigger)(nil)
	_ trigger.TriggerProvider = (*IssueTrigger)(nil)
	_ trigger.SampleProvider  = (*IssueTrigger)(nil)
)

func init() {
	workflow.RegistryNodeMeta(&IssueTrigger{})
}

type IssueTrigger struct {
	jira.Issue
}

type IssueTriggerScope string

const (
	ScopeNewIssue     IssueTriggerScope = "New issue"
	ScopeUpdatedIssue IssueTriggerScope = "Updated issue"
)

type IssueTriggerConfig struct {
	ProjectKey string `json:"projectKey"`
	// choose one: New issue, Updated issue
	Scope IssueTriggerScope `json:"scope"`
}

func (t *IssueTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("issueTrigger"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(IssueTrigger)
		},
		InputForm: spec.InputSchema,
	}
}

func (t *IssueTrigger) Run(c *workflow.NodeContext) (output any, err error) {
	return t.Issue, nil
}

func (t *IssueTrigger) GetConfigObject() any {
	return &IssueTriggerConfig{}
}

// IssueTriggerData represents the form of data stored in
// Data field of the record in table triggers.
type IssueTriggerData struct {
	// Is the trigger enabled?
	Enabled bool `json:"enabled"`
	// The time at and before which all the issues has been processed.
	//
	// When the trigger is enabled, this field should be initialized as Time.Now()
	Progress time.Time `json:"progress"`
	// When did last polling happen?
	LastPolledAt time.Time `json:"lastPolledAt"`
	// Counter
	Successes int `json:"successes"`
	// Counter
	Fails int `json:"fails"`
	// Detail of last error
	LastError string `json:"lastError"`
	// When did the last error happen?
	LastErrorHappenedAt time.Time `json:"lastErrorHappenedAt"`
}

func (t *IssueTrigger) Create(c trigger.WebhookContext) (data map[string]any, err error) {
	// validate user API token
	authorizer := c.GetAuthorizer()

	var credential Credential
	err = authorizer.DecodeMeta(&credential)
	if err != nil {
		err = fmt.Errorf("decoding meta: %w", err)
		return
	}

	client, err := jira.NewClient(jira.Config{
		AccountEmail: credential.AccountEmail,
		APIToken:     credential.APIToken,
		BaseURL:      credential.BaseURL,
	})
	if err != nil {
		err = fmt.Errorf("init Jira client: %w", err)
		return
	}

	_, err = client.GetCurrentUser(c.Context())
	if err != nil {
		err = fmt.Errorf("validating user credential: %w", err)
		return
	}

	triggerData := IssueTriggerData{
		Enabled:  true,
		Progress: time.Now(),
	}

	data, err = utils.ConvertStructToMap(triggerData)
	if err != nil {
		err = fmt.Errorf("ConvertStructToMap: %w", err)
		return
	}

	return
}

func (t *IssueTrigger) Delete(c trigger.WebhookContext) error {
	// relax
	return nil
}

func (t *IssueTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	client, err := newClientFromAuthorizer(c.GetAuthorizer())
	if err != nil {
		err = fmt.Errorf("newClientFromAuthorizer: %w", err)
		return
	}

	config, ok := c.GetConfigObject().(*IssueTriggerConfig)
	if !ok {
		err = fmt.Errorf("unexpected config object, got %T, want *IssueTriggerConfig", c.GetConfigObject())
		return
	}

	jqlQuery := fmt.Sprintf(`project = "%s" order by updated desc`, config.ProjectKey)
	resp, err := client.IssueSearch(c.Context(), jira.IssueSearchOpt{
		JQL:        jqlQuery,
		MaxResults: 10,
	})
	if err != nil {
		err = fmt.Errorf("featching example issues from Jira: %w", err)
		return
	}

	result = make([]trigger.SampleData, len(resp.Issues))
	for i, item := range resp.Issues {
		result[i] = sampleIssue(item)
	}

	return
}

type sampleIssue jira.Issue

func (s sampleIssue) GetID() string {
	return s.ID
}

func (s sampleIssue) GetVersion() string {
	return s.Fields.UpdatedAt.Time().String()
}
