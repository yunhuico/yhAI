package dingtalk

import (
	"errors"
	"fmt"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var (
	ConversationTypeKey = "conversationType"
	CorpID              = "corpID"
)

type CorpBotMessageTrigger struct {
	ConversationID string `json:"conversationId"`
	AtUsers        []struct {
		DingtalkID string `json:"dingtalkId"`
	} `json:"atUsers"`
	ChatbotCorpID             string `json:"chatbotCorpId"`
	ChatbotUserID             string `json:"chatbotUserId"`
	MsgID                     string `json:"msgId"`
	SenderNick                string `json:"senderNick"`
	IsAdmin                   bool   `json:"isAdmin"`
	SenderStaffID             string `json:"senderStaffId"`
	SessionWebhookExpiredTime int64  `json:"sessionWebhookExpiredTime"`
	CreateAt                  int64  `json:"createAt"`
	SenderCorpID              string `json:"senderCorpId"`
	ConversationType          string `json:"conversationType"`
	SenderID                  string `json:"senderId"`
	ConversationTitle         string `json:"conversationTitle"`
	IsInAtList                bool   `json:"isInAtList"`
	SessionWebhook            string `json:"sessionWebhook"`
	Text                      struct {
		Content string `json:"content"`
	} `json:"text"`
	RobotCode string `json:"robotCode"`
	Msgtype   string `json:"msgtype"`
}

type CorpBotMessageTriggerConfig struct {
	ConversationType string `json:"conversationType"`
}

func (s *CorpBotMessageTrigger) GetConfigObject() any {
	return &CorpBotMessageTriggerConfig{}
}

type CorpBotCredentialMetadata struct {
	CorpID string `json:"corpId"`
}

func (s *CorpBotMessageTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	config, ok := c.GetConfigObject().(*CorpBotMessageTriggerConfig)
	if !ok {
		return nil, fmt.Errorf("corp bot message trigger config is invalid")
	}

	meta := &CorpBotCredentialMetadata{}
	err := c.GetAuthorizer().DecodeMeta(meta)
	if err != nil {
		return nil, fmt.Errorf("decode credential metadata: %w", err)
	}

	if meta.CorpID == "" {
		return nil, errors.New("credential corpId required")
	}

	c.SetTriggerQueryID(meta.CorpID)

	return map[string]any{
		ConversationTypeKey: config.ConversationType,
		CorpID:              meta.CorpID,
	}, nil
}

// Delete no resource that needs to delete.
func (s *CorpBotMessageTrigger) Delete(c trigger.WebhookContext) error {
	return nil
}

var _ trigger.TriggerProvider = (*CorpBotMessageTrigger)(nil)

func init() {
	workflow.RegistryNodeMeta(&CorpBotMessageTrigger{})
}

func (s *CorpBotMessageTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalkCorpBot#newMessage")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CorpBotMessageTrigger)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *CorpBotMessageTrigger) Run(c *workflow.NodeContext) (any, error) {
	s.Text.Content = strings.TrimSpace(s.Text.Content)
	return s, nil
}
