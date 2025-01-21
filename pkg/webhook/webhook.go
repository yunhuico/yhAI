package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var webhookHTTPMethods = [...]string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
}

var ok = httpbase.OK

type WorkProducer interface {
	Produce(ctx context.Context, work *work.Work) (err error)
}

type WebhookTriggerStore interface {
	GetTriggerWithNodeByID(ctx context.Context, id string) (trigger model.TriggerWithNode, err error)
}

type WebhookTrigger struct {
	timeoutPerRequest time.Duration

	db       WebhookTriggerStore
	producer WorkProducer

	server *httpbase.GracefulServer
	// mainly for test convenience
	router *gin.Engine

	slackWebhook      *SlackWebhook
	salesforceWebhook *SalesforceWebhook
	dingtalkWebhook   *DingtalkWebhook
}

type WebhookTriggerConfig struct {
	Port int
	// timeout for webhook request handling
	TimeoutSecondsPerRequest int
}

type WebhookTriggerOpt struct {
	WebhookTriggerConfig

	DB       WebhookTriggerStore
	Logger   log.Logger
	Producer WorkProducer
}

func NewWebhookTrigger(opt WebhookTriggerOpt, slackOpt SlackWebhookOpt, salesforceOpt SalesforceWebhookOpt, dingtalkOpt DingtalkCorpBotWebhookOpt) (w *WebhookTrigger, err error) {
	const maxRequestBodySize = 2 << 20 // 2 MiB

	if opt.TimeoutSecondsPerRequest <= 0 {
		opt.TimeoutSecondsPerRequest = 10
	}

	w = &WebhookTrigger{
		timeoutPerRequest: time.Duration(opt.TimeoutSecondsPerRequest) * time.Second,
		db:                opt.DB,
		producer:          opt.Producer,
	}

	middleware := httpbase.Middleware{
		Logger:        opt.Logger,
		AuditResponse: false,
	}

	router := gin.New()
	router.ForwardedByClientIP = true
	router.HandleMethodNotAllowed = true

	router.NoMethod(middleware.MethodNotAllowedHandler)
	router.NoRoute(middleware.NotFoundHandler)

	router.Use(
		middleware.Recovery,
		middleware.LimitRequestBody(maxRequestBodySize),
		middleware.RequestLog,
		middleware.Error,
	)

	router.GET("/healthz", middleware.HelloHandler)
	for _, method := range webhookHTTPMethods {
		router.Handle(method, "/hooks/:webhookID", w.handleWebhook)
	}

	w.slackWebhook, err = NewSlackWebhook(router, slackOpt)
	if err != nil {
		err = fmt.Errorf("creating slack webhook: %w", err)
		return
	}

	w.salesforceWebhook = NewSalesforceWebhook(router, salesforceOpt)
	w.dingtalkWebhook = NewDingtalkWebhook(router, dingtalkOpt)

	w.router = router
	w.server = httpbase.NewGracefulServer(httpbase.GraceServerOpt{
		Logger: opt.Logger,
		Port:   opt.Port,
	}, w.router)

	return
}

func (w *WebhookTrigger) ListenAndServe() error {
	return w.server.ListenAndServe()
}

func (w *WebhookTrigger) Shutdown(ctx context.Context) error {
	return w.server.Shutdown(ctx)
}

func (w *WebhookTrigger) handleWebhook(c *gin.Context) {
	var (
		err         error
		ctx, cancel = context.WithTimeout(c.Request.Context(), w.timeoutPerRequest)
	)
	defer func() {
		cancel()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		ok(c, nil)
	}()

	webhookID := c.Param("webhookID")
	if webhookID == "" {
		err = errInvalidWebhookID
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

	if trigger.Node == nil {
		err = fmt.Errorf("trigger %q node not found", trigger.ID)
		_ = c.Error(err)
		err = errInvalidTrigger
		return
	}

	httpRequest, err := workflow.BuildHTTPRequest(c.Request)
	if err != nil {
		err = fmt.Errorf("building HTTPRequest: %w", err)
		_ = c.Error(err)
		err = errInvalidWebhookCall
		return
	}

	data, err := httpRequest.Marshal()
	if err != nil {
		err = fmt.Errorf("marshing HTTPRequest: %w", err)
		_ = c.Error(err)
		err = errInvalidWebhookCall
		return
	}

	shouldAbort, err := w.preFilterRequest(c, trigger, data)
	if err != nil {
		err = fmt.Errorf("pre filter request: %w", err)
		return
	}
	if shouldAbort {
		return
	}

	workload := work.Work{
		WorkflowID:       trigger.WorkflowID,
		StartNodeID:      trigger.NodeID,
		StartNodePayload: data,
	}
	err = w.producer.Produce(ctx, &workload)
	if err != nil {
		err = fmt.Errorf("pushing workload: %w", err)
		_ = c.Error(err)
		err = errInvalidWebhookCall
		return
	}
}

func (w *WebhookTrigger) preFilterRequest(c *gin.Context, trigger model.TriggerWithNode, data []byte) (shouldAbort bool, err error) {
	nodeMeta, ok := workflow.GetNodeMeta(trigger.Node.Class)
	if !ok {
		err = fmt.Errorf("unknown class %q", trigger.Node.Class)
		return
	}

	filter, ok := nodeMeta.New().(workflow.PreFilterProvider)
	if ok {
		configObj := filter.GetConfigObject()
		if configObj != nil {
			var decoder *mapstructure.Decoder
			decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				Squash: true,
				Result: configObj,
			})
			if err != nil {
				err = fmt.Errorf("initializing decoder: %w", err)
				return
			}

			err = decoder.Decode(trigger.Node.Data.InputFields)
			if err != nil {
				err = fmt.Errorf("bind node input fields to trigger config object: %w", err)
				_ = c.Error(err)
				err = errInvalidTrigger
				return
			}
		}

		shouldAbort, err = filter.PreFilter(configObj, data)
		if err != nil {
			err = fmt.Errorf("preFilter http request: %w", err)
			_ = c.Error(err)
			err = errInvalidWebhookCall
			return
		}
	}

	return
}
