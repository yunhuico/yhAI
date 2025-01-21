package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type (
	SalesforceWebhook struct {
		delayTrigger      *DelayTrigger
		timeoutPerRequest time.Duration
		db                WebhookTriggerStore
	}

	SalesforceWebhookOpt struct {
		DB                WebhookTriggerStore
		TimeoutPerRequest int
		Cache             *cache.Cache
		Ctx               context.Context
		Producer          WorkProducer
	}
)

func NewSalesforceWebhook(router *gin.Engine, opt SalesforceWebhookOpt) *SalesforceWebhook {
	delay := NewDelayTrigger(opt.Ctx, opt.Cache, opt.Producer)
	delay.Start()
	wb := &SalesforceWebhook{
		delayTrigger:      delay,
		db:                opt.DB,
		timeoutPerRequest: time.Duration(opt.TimeoutPerRequest) * time.Second,
	}

	router.POST("/salesforce/hooks/:webhookID/:sobjectID", wb.handleEvents)

	return wb
}

func (w *SalesforceWebhook) handleEvents(c *gin.Context) {
	var (
		err         error
		ctx, cancel = context.WithTimeout(c.Request.Context(), w.timeoutPerRequest)
	)
	defer func() {
		cancel()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	webhookID := c.Param("webhookID")
	sobjectID := c.Param("sobjectID")
	if webhookID == "" {
		err = errInvalidWebhookID
		return
	}
	if sobjectID == "" {
		err = errInvalidSObject
		return
	}

	trigger, err := w.db.GetTriggerWithNodeByID(ctx, webhookID)
	if err != nil {
		err = fmt.Errorf("querying webhook trigger by id %q: %w", webhookID, err)
		_ = c.Error(err)
		// conceal the real error
		err = errInvalidWebhookCall
		return
	}

	httpRequest, err := workflow.BuildHTTPRequest(c.Request)
	if err != nil {
		err = fmt.Errorf("building HTTPRequest: %w", err)
		_ = c.Error(err)
		err = errInvalidWebhookCall
		return
	}

	key := fmt.Sprintf("%s:%s:%s:%s", webhookID, sobjectID, trigger.WorkflowID, trigger.NodeID)
	err = w.delayTrigger.Push(ctx, key, httpRequest)
	if err != nil {
		err = fmt.Errorf("pushing httpRequest: %w", err)
		_ = c.Error(err)
		err = errInvalidWebhookCall
		return
	}

	ok(c, nil)
}
