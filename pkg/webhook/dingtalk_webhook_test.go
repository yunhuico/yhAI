package webhook

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func TestDingtalkWebhook(t *testing.T) {
	t.Run("test verify failed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)
		webhook := NewDingtalkWebhook(router, DingtalkCorpBotWebhookOpt{
			TimeoutPerRequest: 10,
			Ctx:               context.Background(),
			Credential: DingtalkBotCredential{
				AppKey:    "appkey",
				AppSecret: "appSecret",
			},
		})
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer(nil)),
		}
		webhook.handle(c)
		assert.Len(t, c.Errors, 1)
	})

	t.Run("test invalid payload", func(t *testing.T) {
		c, webhook := setupDingtalkWebhook()
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(``))),
		}
		webhook.handle(c)
		assert.Len(t, c.Errors, 1)
	})

	t.Run("test Msgtype is not text, ignore this request", func(t *testing.T) {
		c, webhook := setupDingtalkWebhook()
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"msgtype": "other"}`))),
		}
		webhook.handle(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test query triggers error", func(t *testing.T) {
		c, webhook := setupDingtalkWebhook()
		mockErr := errors.New("mock error")
		webhook.db = &mockWebhookStore{
			err: mockErr,
		}
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"msgtype": "text", "senderCorpId": "copr-id"}`))),
		}
		webhook.handle(c)
		assert.Len(t, c.Errors, 1)
		assert.ErrorIs(t, c.Errors[0], mockErr)
	})

	t.Run("test query empty triggers", func(t *testing.T) {
		c, webhook := setupDingtalkWebhook()
		webhook.db = &mockWebhookStore{
			triggers: nil,
		}
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"msgtype": "text", "senderCorpId": "copr-id"}`))),
		}
		webhook.handle(c)
		assert.Len(t, c.Errors, 0)
	})

	t.Run("test handle successfully", func(t *testing.T) {
		c, webhook := setupDingtalkWebhook()
		mockProducer := &mockWriteOnlyProducer{}
		webhook.producer = mockProducer
		webhook.db = &mockWebhookStore{
			triggers: model.Triggers{
				{
					Data: map[string]any{
						"conversationType": "",
					},
				},
				{
					Data: map[string]any{
						"conversationType": "2",
					},
				},
				{
					Data: map[string]any{
						"conversationType": "1",
					},
				},
				{
					Data: map[string]any{
						"conversationType": "1",
					},
				},
			},
		}
		c.Request = &http.Request{
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{
				"msgtype": "text",
				"senderCorpId": "copr-id",
				"conversationType": "1"
			}`))),
		}
		webhook.handle(c)
		assert.Len(t, c.Errors, 0)
		assert.Len(t, mockProducer.works, 2)
	})
}

func setupDingtalkWebhook() (*gin.Context, *DingtalkWebhook) {
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)
	webhook := NewDingtalkWebhook(router, DingtalkCorpBotWebhookOpt{
		TimeoutPerRequest: 10,
		Ctx:               context.Background(),
		Credential: DingtalkBotCredential{
			AppKey:    "appkey",
			AppSecret: "appSecret",
		},
	})
	webhook.skipSignVerification = true
	return c, webhook
}
