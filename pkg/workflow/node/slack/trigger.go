package slack

import (
	"errors"
	"fmt"

	"github.com/slack-go/slack"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

const (
	TargetSlackEvent = "targetSlackEvent"
	SlackChannelID   = "channelID"
)

var (
	_ trigger.TriggerProvider = (*NewMessageTrigger)(nil)
	_ trigger.SampleProvider  = (*NewMessageTrigger)(nil)
)

type NewFileTrigger struct{}

func (t *NewFileTrigger) Run(c *workflow.NodeContext) (any, error) {
	// TODO implement me
	panic("implement me")
}

func (t *NewFileTrigger) UltrafoxNode() workflow.NodeMeta {
	return workflow.NodeMeta{
		Class: "ultrafox/slack#channelTopic",
		New: func() workflow.Node {
			return new(NewFileTrigger)
		},
	}
}

type NewMessageTrigger struct {
	EventType string `json:"type"`
	Channel   string `json:"channel"`
	User      string `json:"user"`
	Text      string `json:"text"`
	TimeStamp string `json:"ts"`
}

type slackMeta struct {
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
}

func getSlackTeamID(c trigger.WebhookContext) (string, error) {
	meta := &slackMeta{}
	err := c.GetAuthorizer().DecodeTokenMetaData(c.Context(), meta)
	if err != nil {
		return "", err
	}
	return meta.Team.ID, nil
}

func (t *NewMessageTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	config, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("message trigger config is invalid")
	}
	channelID := config.ChannelID
	if channelID == "" {
		return nil, fmt.Errorf("channelID is empty")
	}

	teamID, err := getSlackTeamID(c)
	if err != nil {
		return nil, fmt.Errorf("decoding slack meta: %w", err)
	}
	c.SetTriggerQueryID(teamID)

	return map[string]any{
		TargetSlackEvent: "message",
		SlackChannelID:   channelID,
	}, nil
}

func (t *NewMessageTrigger) Delete(c trigger.WebhookContext) error {
	c.SetTriggerQueryID("")

	return nil
}

func (t *NewMessageTrigger) UltrafoxNode() workflow.NodeMeta {
	return workflow.NodeMeta{
		Class: "ultrafox/slack#triggerMessage",
		New: func() workflow.Node {
			return new(NewMessageTrigger)
		},
	}
}

func (t *NewMessageTrigger) Run(c *workflow.NodeContext) (any, error) {
	if t.Channel == "" || t.TimeStamp == "" {
		return nil, errors.New("unexpected message, channel or timestamp is empty")
	}
	return MessageData{
		Message:   t.Text,
		ChannelID: t.Channel,
		Timestamp: t.TimeStamp,
	}, nil
}

func (t *NewMessageTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	config, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		err = fmt.Errorf("message trigger config is invalid")
		return
	}
	channelID := config.ChannelID
	if channelID == "" {
		err = fmt.Errorf("channelID is empty")
		return
	}

	client, err := newClient(c.Context(), c.GetAuthorizer())
	if err != nil {
		err = fmt.Errorf("get sample list: %w", err)
		return
	}

	resp, err := client.GetConversationHistoryContext(c.Context(), &slack.GetConversationHistoryParameters{
		Limit:     10,
		ChannelID: channelID,
	})
	if err != nil {
		err = fmt.Errorf("slack get conversation history: %w", err)
		return
	}

	if !resp.Ok {
		err = fmt.Errorf("slack conversation response is not ok")
		return
	}

	for i := len(resp.Messages) - 1; i >= 0; i-- {
		msg := resp.Messages[i]
		if msg.Type != slack.TYPE_MESSAGE {
			continue
		}

		// filter the empty message
		if msg.Text == "" {
			continue
		}

		// filter the bot message
		if msg.BotID != "" {
			continue
		}

		result = append(result, &MessageData{
			Message:   msg.Text,
			ChannelID: channelID,
			Timestamp: msg.Timestamp,
		})
	}

	return
}

var _ trigger.SampleData = (*MessageData)(nil)

type TriggerConfig struct {
	ChannelID string `json:"channelId"`
}

func (t *NewMessageTrigger) GetConfigObject() any {
	return &TriggerConfig{}
}

type MessageData struct {
	Message   string `json:"message"`
	ChannelID string `json:"channelId"`
	Timestamp string `json:"timestamp"`
}

func (m MessageData) GetID() string {
	return m.Timestamp
}

func (m MessageData) GetVersion() string {
	return m.Timestamp
}

type Manifest struct {
	DisplayInformation struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		LongDescription string `json:"long_description"`
		BackgroundColor string `json:"background_color"`
	} `json:"display_information"`
	Settings struct {
		AllowedIPAddressRanges []string `json:"allowed_ip_address_ranges"`
		EventSubscriptions     struct {
			RequestURL string   `json:"request_url"`
			BotEvents  []string `json:"bot_events"`
			UserEvents []string `json:"user_events"`
		} `json:"event_subscriptions"`
	} `json:"settings"`
	Features    struct{} `json:"features"`
	OAuthConfig struct {
		Scopes struct {
			Bot  []string `json:"bot"`
			User []string `json:"user"`
		} `json:"scopes"`
	} `json:"oauth_config"`
}

type AppMetaData struct {
	ApplicationID string `json:"app_id"`
	Credentials   struct {
		ClientID          string `json:"client_id"`
		ClientSecret      string `json:"client_secret"`
		VerificationToken string `json:"verification_token"`
		SigningSecret     string `json:"signing_secret"`
	}
	OauthAuthorizeUrl string `json:"oauth_authorize_url"`
}
