package dingtalk

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var _ workflow.Node = (*ReplyMessage)(nil)

type ReplyMessage struct {
	SessionWebhook string   `json:"sessionWebhook"`
	MessageType    string   `json:"messageType"`
	AtUserID       string   `json:"atUserId"`
	Content        string   `json:"content"`
	Actions        []string `json:"actions"`
}

func init() {
	workflow.RegistryNodeMeta(&ReplyMessage{})
}

func (s *ReplyMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalkCorpBot#replyMessage")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ReplyMessage)
		},
		InputForm: spec.InputSchema,
	}
}

const dtmdFormat = "[%s](dtmd://dingtalkclient/sendMessage?content=%s)"

func (s *ReplyMessage) Run(c *workflow.NodeContext) (output any, err error) {
	if s.Content == "" {
		return nil, errors.New("content is required")
	}

	var message *DingTalkMessage
	if s.MessageType == "text" {
		message = &DingTalkMessage{
			MessageType: "text",
			TextMessage: TextMessage{
				Content: s.Content,
			},
			At: At{
				AtUserIDs: []string{s.AtUserID},
			},
		}
	} else if s.MessageType == "markdown" {
		message = &DingTalkMessage{
			MessageType: "markdown",
			MarkdownMessage: MarkdownMessage{
				Title: "reply",
				Text:  s.Content,
			},
			At: At{
				AtUserIDs: []string{s.AtUserID},
			},
		}
	} else if s.MessageType == "actionText" {
		text := s.Content
		for _, action := range s.Actions {
			text += "\n > " + fmt.Sprintf(dtmdFormat, action, url.QueryEscape(action)) + "\n"
		}
		message = &DingTalkMessage{
			MessageType: "markdown",
			MarkdownMessage: MarkdownMessage{
				Title: s.Content,
				Text:  text,
			},
			At: At{
				AtUserIDs: []string{s.AtUserID},
			},
		}
	}

	r := message.toReader()
	req, _ := http.NewRequest("POST", s.SessionWebhook, r)
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("reply message: %w", err)
		return
	}

	response, err := handleResponse(resp)
	if err != nil {
		return nil, err
	}

	if response.ErrCode != SendSuccessStatus {
		return nil, fmt.Errorf("send message: %s", response.ErrMsg)
	}

	return map[string]any{
		"success": true,
	}, nil
}
