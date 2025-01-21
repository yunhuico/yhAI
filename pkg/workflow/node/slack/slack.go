package slack

import (
	"context"
	"embed"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"

	"github.com/slack-go/slack"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapterMeta := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapterMeta.RegisterSpecsByDir(adapterDir)

	adapterMeta.RegisterCredentialTemplate(adapter.OAuth2CredentialType, `{
	"metaData": {
		"server": "{{ .server }}",
		"extraKeys": ["team", "authed_user", "app_id", "bot_user_id"]
	},
	"oauth2Config": {
		"clientId": "{{ .clientId }}",
		"clientSecret": "{{ .clientSecret }}",
		"redirectUrl": "{{ .redirectUrl }}",
		"scopes": [
			"channels:history",
			"channels:read",
			"groups:history",
			"groups:read",
			"im:history",
			"im:read",
			"mpim:history",
			"mpim:read",
			"reactions:read",
			"reactions:write",
			"chat:write",
			"pins:read",
			"pins:write",
			"im:write",
			"users.profile:read",
			"users:read",
			"users:read.email"
		],
		"endpoint": {
			"authUrl": "{{ .server }}/oauth/v2/authorize",
			"tokenUrl": "{{ .server }}/api/oauth.v2.access"
		}
	}
}`)

	workflow.RegistryNodeMeta(&NewMessageTrigger{})

	workflow.RegistryNodeMeta(&DirectMessage{})
	workflow.RegistryNodeMeta(&ChannelMessage{})
	workflow.RegistryNodeMeta(&ChannelTopic{})
	workflow.RegistryNodeMeta(&ChannelAddPin{})
	workflow.RegistryNodeMeta(&ChannelRemovePin{})
	workflow.RegistryNodeMeta(&AddReaction{})
	workflow.RegistryNodeMeta(&ListChannels{})
	workflow.RegistryNodeMeta(&ListUsers{})
}

type BaseSlackNode struct {
	client *slack.Client
}

func (s *BaseSlackNode) Provision(ctx context.Context, dependencies workflow.ProvisionDeps) error {
	client, err := newClient(ctx, dependencies.Authorizer)

	if err != nil {
		return err
	}
	s.client = client
	return nil
}

func (s *BaseSlackNode) GetClient() *slack.Client {
	return s.client
}

func newClient(ctx context.Context, sign auth.Authorizer) (*slack.Client, error) {
	accessToken, err := sign.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var client *slack.Client
	if sign.CredentialType() == model.CredentialTypeAccessToken {
		client = slack.New(accessToken)
	} else if sign.CredentialType() == model.CredentialTypeOAuth2 {
		client = slack.New(accessToken)
	}
	return client, nil
}
