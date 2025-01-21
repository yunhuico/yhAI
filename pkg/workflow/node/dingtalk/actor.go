package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type At struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
	AtUserIDs []string `json:"atUserIds,omitempty"`
}

type Link struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageURL string `json:"messageUrl"`
	PictureURL string `json:"picUrl"`
}

type Btn struct {
	Title     string `json:"title"`
	ActionURL string `json:"actionUrl"`
}

type SendMarkdownMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`

	// Optional
	At
}

type SendTextMessage struct {
	Content string `json:"content"`

	// Optional
	At
}

type SendLinkMessage struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageURL string `json:"messageUrl"`

	// Optional
	PictureURL string `json:"picUrl,omitempty"`
}

type SendActionCardMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`

	SingleTitle string `json:"singleTitle,omitempty"`
	SingleURL   string `json:"singleUrl,omitempty"`

	Btns []Btn `json:"btns,omitempty"`

	// Optional
	BtnOrientation string `json:"btnOrientation,omitempty"`
}

type SendFeedCardMessage struct {
	Links []Link `json:"links"`
}

func (s *SendTextMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalk#sendTextMessage")
	return workflow.NodeMeta{
		Class: "ultrafox/dingtalk#sendTextMessage",
		New: func() workflow.Node {
			return new(SendTextMessage)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *SendMarkdownMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalk#sendMarkdownMessage")
	return workflow.NodeMeta{
		Class: "ultrafox/dingtalk#sendMarkdownMessage",
		New: func() workflow.Node {
			return new(SendMarkdownMessage)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *SendLinkMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalk#sendLinkMessage")
	return workflow.NodeMeta{
		Class: "ultrafox/dingtalk#sendLinkMessage",
		New: func() workflow.Node {
			return new(SendLinkMessage)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *SendFeedCardMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalk#sendFeedCardMessage")
	return workflow.NodeMeta{
		Class: "ultrafox/dingtalk#sendFeedCardMessage",
		New: func() workflow.Node {
			return new(SendFeedCardMessage)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *SendActionCardMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalk#sendActionCardMessage")
	return workflow.NodeMeta{
		Class: "ultrafox/dingtalk#sendActionCardMessage",
		New: func() workflow.Node {
			return new(SendActionCardMessage)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *SendTextMessage) Run(ctx *workflow.NodeContext) (any, error) {
	message := &DingTalkMessage{
		MessageType: "text",
		TextMessage: TextMessage{
			Content: s.Content,
		},
		At: s.At,
	}
	return run(ctx, message)
}

func (s *SendActionCardMessage) Run(ctx *workflow.NodeContext) (any, error) {
	if s.Btns == nil && (s.SingleTitle == "" || s.SingleURL == "") {
		return nil, errors.New("btns or singleURL should either be filled")
	}
	message := &DingTalkMessage{
		MessageType:       "actionCard",
		ActionCardMessage: ActionCardMessage(*s),
	}
	return run(ctx, message)
}

func (s *SendFeedCardMessage) Run(ctx *workflow.NodeContext) (any, error) {
	if s.Links == nil {
		return nil, errors.New("links is nil")
	}

	message := &DingTalkMessage{
		MessageType: "feedCard",
	}

	for _, link := range s.Links {
		if link.PictureURL == "" {
			return nil, errors.New("picture URL should not be empty")
		}
		message.FeedCardMessage.Links = append(message.FeedCardMessage.Links, FeedCardLink(link))
	}

	return run(ctx, message)
}

func (s *SendLinkMessage) Run(ctx *workflow.NodeContext) (any, error) {
	message := &DingTalkMessage{
		MessageType: "link",
		LinkMessage: LinkMessage(*s),
	}
	return run(ctx, message)
}

func (s *SendMarkdownMessage) Run(ctx *workflow.NodeContext) (any, error) {
	message := &DingTalkMessage{
		MessageType: "markdown",
		MarkdownMessage: MarkdownMessage{
			Title: s.Title,
			Text:  s.Text,
		},
		At: s.At,
	}
	return run(ctx, message)
}

func run(ctx *workflow.NodeContext, message *DingTalkMessage) (output any, err error) {
	authorizer := ctx.GetAuthorizer()
	token, err := authorizer.GetAccessToken(ctx.Context())
	if err != nil {
		err = fmt.Errorf("get access token: %w", err)
		return
	}

	meta := &dingTalkAuthMeta{}
	err = authorizer.DecodeMeta(meta)
	if err != nil {
		err = fmt.Errorf("decoding ding talk secret: %w", err)
		return
	}

	webhookUrl, err := makeWebhookUrl(token, meta.Secret)
	if err != nil {
		err = fmt.Errorf("create webhookUrl for dingtalk: %w", err)
		return
	}

	err = sendWebhook(ctx.Context(), webhookUrl, message)
	if err != nil {
		err = fmt.Errorf("sending webhook: %w", err)
		return
	}

	output = map[string]any{
		"success": true,
	}
	return
}

func sendWebhook(ctx context.Context, webhookUrl string, message *DingTalkMessage) (err error) {
	req, err := http.NewRequest(http.MethodPost, webhookUrl, message.toReader())
	if err != nil {
		err = fmt.Errorf("initializing HTTP request: %w", err)
		return
	}

	req.Header.Set("Content-type", "application/json")
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("performing HTTP request: %w", err)
		return
	}
	defer resp.Body.Close()

	response, err := handleResponse(resp)
	if err != nil {
		err = fmt.Errorf("handling response: %w", err)
		return
	}

	if response.ErrCode != SendSuccessStatus {
		err = fmt.Errorf("response code is't success: %s", response.ErrMsg)
		return
	}

	return
}

func handleResponse(resp *http.Response) (*DingResponse, error) {
	detail, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dingtalk API response with code %d: %s", resp.StatusCode, detail)
	}

	response := &DingResponse{}
	err := json.Unmarshal(detail, &response)
	if err != nil {
		return nil, fmt.Errorf("read response for ding talk: %w", err)
	}

	if response.ErrCode != SendSuccessStatus {
		return nil, fmt.Errorf("send message: %s", response.ErrMsg)
	}

	return response, nil
}

type ActionCardMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`

	SingleTitle string `json:"singleTitle,omitempty"`
	SingleURL   string `json:"singleURL,omitempty"`

	Btns []Btn `json:"btns,omitempty"`

	// Optional
	BtnOrientation string `json:"btnOrientation,omitempty"`
}

type LinkMessage struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageURL string `json:"messageUrl"`

	// Optional
	PictureURL string `json:"picUrl,omitempty"`
}

type FeedCardLink struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageURL string `json:"messageURL"`
	PictureURL string `json:"picURL"`
}

type FeedCardMessage struct {
	Links []FeedCardLink `json:"links"`
}

type MarkdownMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type TextMessage struct {
	Content string `json:"content"`
}

type SendMessage struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`

	At
}

func (s *SendMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/dingtalk#sendMessage")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(SendMessage)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *SendMessage) Run(ctx *workflow.NodeContext) (any, error) {
	message := &DingTalkMessage{
		MessageType: s.Type,
		At:          s.At,
	}
	switch s.Type {
	case "markdown":
		message.MarkdownMessage = MarkdownMessage{
			Title: s.Title,
			Text:  s.Content,
		}
	case "text":
		fallthrough // default is text
	default:
		message.MessageType = "text"
		message.TextMessage = TextMessage{
			Content: s.Content,
		}
	}

	return run(ctx, message)
}

type DingTalkMessage struct {
	MessageType       string `json:"msgtype"`
	LinkMessage       `json:"link,omitempty"`
	FeedCardMessage   `json:"feedCard,omitempty"`
	ActionCardMessage `json:"actionCard,omitempty"`
	MarkdownMessage   `json:"markdown,omitempty"`
	TextMessage       `json:"text,omitempty"`
	At                `json:"at,omitempty"`
}

func (m DingTalkMessage) toReader() io.Reader {
	b, _ := json.Marshal(m)
	return bytes.NewReader(b)
}
