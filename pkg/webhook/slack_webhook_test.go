package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
)

func TestNewSlackWebhook(t *testing.T) {
	_, err := NewSlackWebhook(nil, SlackWebhookOpt{})
	assert.Error(t, err)
}

func TestSlackHandleEvent(t *testing.T) {
	t.Run("test verify failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer(nil)),
		}
		slackWebhook, err := NewSlackWebhook(router, SlackWebhookOpt{
			TimeoutSecondsPerRequest: 10,
			SkipSignVerification:     false,
			SigningSecret:            "xxxxx",
		})
		assert.NoError(t, err)

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 2)
		assert.ErrorIs(t, c.Errors[1], errUnauthorized)
	})

	newSlackWebhook := func(router *gin.Engine) (*SlackWebhook, error) {
		return NewSlackWebhook(router, SlackWebhookOpt{
			TimeoutSecondsPerRequest: 10,
			SkipSignVerification:     true,
			SigningSecret:            "xxxxx",
		})
	}

	t.Run("test request body invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer(nil)),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 2)
		assert.ErrorIs(t, c.Errors[1], errInvalidEvent)
	})

	t.Run("test url verification success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"type":"url_verification"}`))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test event type is not message type, ignore this event", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"type":"event_callback", "event": {"type": "app_mention"}}`))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test message type event, but text is empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"type":"event_callback", "event": {"type": "message"}}`))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test message type event, but subType is message_chaned", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"type":"event_callback", "event": {"type": "message", "subtype": "message_changed"}}`))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test message type event, but message sent by bot", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"type":"event_callback", "event": {"type": "message", "bot_id": "bot-id"}}`))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test message type event, but database return error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		body := `{"type":"event_callback", "event": {"type": "message", "text": "hello"}}`
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(body))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)
		mockErr := errors.New("mock error")
		slackWebhook.db = &mockWebhookStore{mockErr, nil}

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 2)
		assert.ErrorIs(t, c.Errors[0], mockErr)
	})

	t.Run("test message type event, but no triggers", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		body := `{"type":"event_callback", "event": {"type": "message", "text": "hello"}}`
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(body))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)
		slackWebhook.db = &mockWebhookStore{nil, nil}

		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test message type event handle successfully", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		channelID := "this-is-a-channel"
		body := fmt.Sprintf(`{"type":"event_callback", "teamID": "1", "event": {"type": "message", "text": "hello", "channel": "%s"}}`, channelID)
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(body))),
		}
		slackWebhook, err := newSlackWebhook(router)
		assert.NoError(t, err)
		slackWebhook.logger, err = log.New("go-test", log.DebugLevel)
		assert.NoError(t, err)
		slackWebhook.db = &mockWebhookStore{nil, model.Triggers{
			{ // 1. no TargetSlackEvent
				Data: map[string]any{},
			},
			{ // 2. no SlackChannelID
				Data: map[string]any{
					TargetSlackEvent: "message",
				},
			},
			{ // 3. matched trigger
				WorkflowID: "workflow-1",
				NodeID:     "workflow-1-node-1",
				Data: map[string]any{
					TargetSlackEvent: "message",
					SlackChannelID:   channelID,
				},
			},
			{ // 4. matched trigger
				WorkflowID: "workflow-2",
				NodeID:     "workflow-2-node-2",
				Data: map[string]any{
					TargetSlackEvent: "message",
					SlackChannelID:   channelID,
				},
			},
			{ // 5. channelID not matched
				WorkflowID: "workflow-2",
				NodeID:     "workflow-2-node-2",
				Data: map[string]any{
					TargetSlackEvent: "message",
					SlackChannelID:   "new-channel-id",
				},
			},
		}}
		mockProducer := &mockWriteOnlyProducer{}
		slackWebhook.producer = mockProducer
		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 0)
		assert.Len(t, mockProducer.works, 2)
		assert.Equal(t, mockProducer.works[0].WorkflowID, "workflow-1")
		assert.Equal(t, mockProducer.works[0].StartNodeID, "workflow-1-node-1")
		assert.Equal(t, mockProducer.works[1].WorkflowID, "workflow-2")
		assert.Equal(t, mockProducer.works[1].StartNodeID, "workflow-2-node-2")

		// test producer error
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(body))),
		}
		mockErr := errors.New("mock error")
		mockProducer = &mockWriteOnlyProducer{err: mockErr}
		slackWebhook.producer = mockProducer
		slackWebhook.HandleEvents(c)
		assert.Len(t, c.Errors, 2)
		assert.ErrorIs(t, c.Errors[0], mockErr)
		assert.ErrorIs(t, c.Errors[1], errInvalidWebhookCall)
	})
}

type mockWebhookStore struct {
	err      error
	triggers model.Triggers
}

func (s mockWebhookStore) GetTriggersByQueryID(ctx context.Context, queryID string) (trigger model.Triggers, err error) {
	return s.triggers, s.err
}

type mockWriteOnlyProducer struct {
	works []*work.Work
	err   error
}

func (p *mockWriteOnlyProducer) Produce(ctx context.Context, work *work.Work) (err error) {
	if p.err != nil {
		return p.err
	}
	p.works = append(p.works, work)
	return
}
