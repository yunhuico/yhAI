package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/slack-go/slack"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type SlackActor struct {
	WebhookURL string `json:"webhookUrl"`
	Msg        string `json:"msg"`
}

func (s *SlackActor) UltrafoxNode() workflow.NodeMeta {
	return workflow.NodeMeta{
		Class: "ultrafox/slack#webhookMessage",
		New: func() workflow.Node {
			return new(SlackActor)
		},
	}
}

func (s *SlackActor) Run(c *workflow.NodeContext) (output any, err error) {
	body := map[string]string{
		"text": s.Msg,
	}
	marshaled, err := json.Marshal(body)
	if err != nil {
		err = fmt.Errorf("marshaling request body: %w", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewReader(marshaled))
	if err != nil {
		err = fmt.Errorf("initializing HTTP request: %w", err)
		return
	}
	req.Header.Set("Content-type", "application/json")
	req = req.WithContext(c.Context())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("performing HTTP request: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("slack API response with code %d: %s", resp.StatusCode, detail)
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil, nil
}

type DirectMessage struct {
	BaseSlackNode `json:"-"`
	UserID        string `json:"userId"`
	Message       string `json:"message"`
}

func (d *DirectMessage) Run(c *workflow.NodeContext) (any, error) {
	userID, timestamp, err := d.client.PostMessageContext(c.Context(), d.UserID, slack.MsgOptionText(d.Message, false))
	if err != nil {
		return nil, fmt.Errorf("slack#directMessage: %w", err)
	}

	return map[string]string{
		"userID":    userID,
		"timestamp": timestamp,
	}, nil
}

func (d *DirectMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#directMessage")
	return workflow.NodeMeta{
		Class: "ultrafox/slack#directMessage",
		New: func() workflow.Node {
			return new(DirectMessage)
		},
		InputForm: spec.InputSchema,
	}
}

type ChannelMessage struct {
	BaseSlackNode `json:"-"`
	ChannelID     string `json:"channelId"`
	Message       string `json:"message"`
}

func (c *ChannelMessage) Run(ctx *workflow.NodeContext) (any, error) {
	channelID, timestamp, err := c.client.PostMessageContext(ctx.Context(), c.ChannelID, slack.MsgOptionText(c.Message, false))
	if err != nil {
		return nil, fmt.Errorf("slack#channelMessage: %w", err)
	}

	return map[string]string{
		"channelID": channelID,
		"timestamp": timestamp,
	}, nil
}

func (c *ChannelMessage) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#channelMessage")
	return workflow.NodeMeta{
		Class: "ultrafox/slack#channelMessage",
		New: func() workflow.Node {
			return new(ChannelMessage)
		},
		InputForm: spec.InputSchema,
	}
}

type ChannelTopic struct {
	BaseSlackNode `json:"-"`
	ChannelID     string `json:"channelId"`
	Topic         string `json:"topic"`
}

func (c *ChannelTopic) Run(ctx *workflow.NodeContext) (any, error) {
	ch, err := c.client.SetTopicOfConversationContext(ctx.Context(), c.ChannelID, c.Topic)
	if err != nil {
		return nil, fmt.Errorf("set topic on channel: %s, error: %w", c.ChannelID, err)
	}
	return ch, nil
}

func (c *ChannelTopic) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#channelTopic")
	return workflow.NodeMeta{
		Class: "ultrafox/slack#channelTopic",
		New: func() workflow.Node {
			return new(ChannelTopic)
		},
		InputForm: spec.InputSchema,
	}
}

type ChannelAddPin struct {
	BaseSlackNode

	// which channel should we operate on
	ChannelID string `json:"channelId"`
	// message timestamp
	Timestamp string `json:"timestamp"`
}

func (p *ChannelAddPin) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#channelAddPin")
	return workflow.NodeMeta{
		Class: "ultrafox/slack#channelAddPin",
		New: func() workflow.Node {
			return new(ChannelAddPin)
		},
		InputForm: spec.InputSchema,
	}
}

func (p *ChannelAddPin) Run(c *workflow.NodeContext) (any, error) {
	err := p.client.AddPinContext(c.Context(), p.ChannelID, slack.ItemRef{
		Timestamp: p.Timestamp,
	})
	if err != nil {
		err = fmt.Errorf("pinning message: %w", err)
		return nil, err
	}

	return nil, nil
}

type ChannelRemovePin struct {
	BaseSlackNode

	// which channel should we operate on
	ChannelID string `json:"channelId"`
	// message timestamp
	Timestamp string `json:"timestamp"`
	// if message is not pinned or non-existed, should there be an error?
	RegardMissingAsError bool `json:"regardMissingAsError"`
}

func (p *ChannelRemovePin) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#channelRemovePin")
	return workflow.NodeMeta{
		Class: "ultrafox/slack#channelRemovePin",
		New: func() workflow.Node {
			return new(ChannelRemovePin)
		},
		InputForm: spec.InputSchema,
	}
}

func (p *ChannelRemovePin) Run(c *workflow.NodeContext) (any, error) {
	err := p.client.RemovePinContext(c.Context(), p.ChannelID, slack.ItemRef{
		Timestamp: p.Timestamp,
	})
	if err == nil {
		return nil, nil
	}
	if p.RegardMissingAsError || !isMissingPin(err) {
		err = fmt.Errorf("un-pinning message: %w", err)
		return nil, err
	}

	return nil, nil
}

type AddReaction struct {
	BaseSlackNode
	ChannelID string   `json:"channelId"`
	Timestamp string   `json:"timestamp"`
	Emojis    []string `json:"emojis"`
}

func (t *AddReaction) Run(ctx *workflow.NodeContext) (any, error) {
	if t.ChannelID == "" || t.Timestamp == "" {
		return nil, errors.New("input for addReaction is empty")
	}

	itemRef := slack.ItemRef{
		Channel:   t.ChannelID,
		Timestamp: t.Timestamp,
	}
	for _, e := range t.Emojis {
		err := t.client.AddReactionContext(ctx.Context(), e, itemRef)
		if err != nil {
			return nil, fmt.Errorf("emoji reaction: %w", err)
		}
	}

	return nil, nil
}

func (t *AddReaction) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#addReaction")
	return workflow.NodeMeta{
		Class: "ultrafox/slack#addReaction",
		New: func() workflow.Node {
			return new(AddReaction)
		},
		InputForm: spec.InputSchema,
	}
}

// missingPinMsgs takes from https://api.slack.com/methods/pins.remove#errors
var missingPinMsgs = map[string]bool{
	"file_not_found":         true,
	"file_comment_not_found": true,
	"message_not_found":      true,
	"not_pinned":             true,
	"no_pin":                 true,
}

func isMissingPin(err error) bool {
	if err == nil {
		return false
	}

	return missingPinMsgs[err.Error()]
}
