package jira

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/jira"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/common"
)

const className = "ultrafox/jira"

var adapterClass = common.AdapterClass(className)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapter := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapter.RegisterSpecsByDir(adapterDir)

	adapter.RegisterCredentialTestingFunc(testCredential)
}

func testCredential(ctx context.Context, credentialType model.CredentialType, fields model.InputFields) (err error) {
	var client *jira.Client
	client, err = jira.NewClient(jira.Config{
		AccountEmail: fields.GetString("accountEmail"),
		APIToken:     fields.GetString("apiToken"),
		BaseURL:      fields.GetString("baseUrl"),
	})
	if err != nil {
		err = fmt.Errorf("init Jira client: %w", err)
		return
	}

	_, err = client.GetInstanceInfo(ctx)

	if err != nil {
		err = fmt.Errorf("testing jira credential: %w", err)
		return
	}

	return
}

type Credential struct {
	// user's Jira account email
	AccountEmail string `json:"accountEmail"`
	// generated at https://id.atlassian.com/manage-profile/security/api-tokens
	APIToken string `json:"apiToken"`
	// e.g. https://your-domain.atlassian.net/ or https://issues.apache.org/jira/
	BaseURL string `json:"baseUrl"`
}

func newClientFromAuthorizer(authorizer auth.Authorizer) (client *jira.Client, err error) {
	if authorizer == nil {
		err = errors.New("provided authorizer is nil")
		return
	}

	var credential Credential
	err = authorizer.DecodeMeta(&credential)
	if err != nil {
		err = fmt.Errorf("decoding meta into credential: %w", err)
		return
	}

	client, err = jira.NewClient(jira.Config{
		AccountEmail: credential.AccountEmail,
		APIToken:     credential.APIToken,
		BaseURL:      credential.BaseURL,
	})
	if err != nil {
		err = fmt.Errorf("init Jira client: %w", err)
		return
	}

	return
}

type baseJiraNode struct {
	client *jira.Client
}

func (b *baseJiraNode) Provision(ctx context.Context, dependencies workflow.ProvisionDeps) (err error) {
	b.client, err = newClientFromAuthorizer(dependencies.Authorizer)
	if err != nil {
		err = fmt.Errorf("newClientFromAuthorizer: %w", err)
		return
	}

	return
}
