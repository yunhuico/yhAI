package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
)

var (
	ConversationTypeKey = "conversationType"
)

type (
	// DingtalkBotCredential
	// reference: https://open.dingtalk.com/document/robots/enterprise-created-chatbot
	DingtalkBotCredential struct {
		AppKey    string
		AppSecret string
	}

	DingtalkCorpBotStore interface {
		GetTriggersByQueryID(ctx context.Context, queryID string) (trigger model.Triggers, err error)
	}

	DingtalkCorpBotWebhookOpt struct {
		DB                DingtalkCorpBotStore
		TimeoutPerRequest int
		Ctx               context.Context
		Producer          WorkProducer
		Credential        DingtalkBotCredential
	}

	DingtalkWebhook struct {
		db                   DingtalkCorpBotStore
		timeoutPerRequest    time.Duration
		ctx                  context.Context
		producer             WorkProducer
		credential           DingtalkBotCredential
		skipSignVerification bool
	}
)

func NewDingtalkWebhook(router *gin.Engine, opt DingtalkCorpBotWebhookOpt) *DingtalkWebhook {
	handler := &DingtalkWebhook{
		db:                opt.DB,
		timeoutPerRequest: time.Duration(opt.TimeoutPerRequest) * time.Second,
		ctx:               opt.Ctx,
		producer:          opt.Producer,
		credential:        opt.Credential,
	}

	router.Handle("POST", "/events/dingtalk", handler.handle)
	return handler
}

func (w DingtalkWebhook) handle(c *gin.Context) {
	var (
		err       error
		ctx       = c.Request.Context()
		_, cancel = context.WithTimeout(ctx, w.timeoutPerRequest)
	)
	defer func() {
		cancel()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		c.String(http.StatusOK, "OK")
	}()

	if err = w.verify(c.Request); err != nil {
		return
	}

	defer c.Request.Body.Close()
	var body []byte
	body, err = io.ReadAll(c.Request.Body)
	if err != nil {
		err = fmt.Errorf("read body: %w", err)
		return
	}

	var event webhookEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		err = fmt.Errorf("json unmarshal error: %w", err)
		return
	}

	if event.Msgtype != "text" {
		return
	}

	triggers, err := w.db.GetTriggersByQueryID(ctx, event.SenderCorpID)
	if err != nil {
		err = fmt.Errorf("querying triggers by query id %q: %w", event.SenderCorpID, err)
		return
	}

	for _, trigger := range triggers {
		conversationType, ok := trigger.Data[ConversationTypeKey].(string)
		if !ok || conversationType != event.ConversationType {
			continue
		}

		workload := work.Work{
			WorkflowID:       trigger.WorkflowID,
			StartNodeID:      trigger.NodeID,
			StartNodePayload: body,
		}
		err = w.producer.Produce(ctx, &workload)
		if err != nil {
			err = fmt.Errorf("pushing workload: %w", err)
			return
		}
	}
}

type webhookEvent struct {
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

func (w DingtalkWebhook) verify(req *http.Request) (err error) {
	if w.skipSignVerification {
		return nil
	}

	headerSign := req.Header.Get("sign")
	headerTimestamp := req.Header.Get("timestamp")

	stringToSign := fmt.Sprintf("%s\n%s", headerTimestamp, w.credential.AppSecret)
	hash := hmac.New(sha256.New, []byte(w.credential.AppSecret))
	hash.Write([]byte(stringToSign))
	signData := hash.Sum(nil)
	sign := base64.StdEncoding.EncodeToString(signData)

	if headerSign == "" || sign == "" {
		err = fmt.Errorf("invalid signature")
		return
	}

	if sign != headerSign {
		err = fmt.Errorf("invalid signature")
		return
	}
	return
}
