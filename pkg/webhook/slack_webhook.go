package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"

	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const (
	TargetSlackEvent = "targetSlackEvent"
	SlackChannelID   = "channelID"
)

type (
	SlackWebhookStore interface {
		GetTriggersByQueryID(ctx context.Context, queryID string) (trigger model.Triggers, err error)
	}

	SlackWebhookOpt struct {
		TimeoutSecondsPerRequest int
		DB                       SlackWebhookStore
		Producer                 WorkProducer
		Logger                   log.Logger

		SigningSecret        string
		SkipSignVerification bool
	}

	SlackWebhook struct {
		timeoutPerRequest time.Duration

		db       SlackWebhookStore
		producer WorkProducer
		logger   log.Logger

		signingSecret string

		// used in testing only
		skipSignVerification bool
	}
)

// NewSlackWebhook creates webhook for handling slack events.
func NewSlackWebhook(router *gin.Engine, opt SlackWebhookOpt) (*SlackWebhook, error) {
	if opt.SigningSecret == "" {
		err := errors.New("initializing slack webhook: no signing secret provided")
		return nil, err
	}

	s := &SlackWebhook{
		timeoutPerRequest:    time.Duration(opt.TimeoutSecondsPerRequest) * time.Second,
		db:                   opt.DB,
		producer:             opt.Producer,
		logger:               opt.Logger,
		signingSecret:        opt.SigningSecret,
		skipSignVerification: opt.SkipSignVerification,
	}
	router.Handle(http.MethodPost, "/events/slack", s.HandleEvents)

	return s, nil
}

func (s *SlackWebhook) HandleEvents(c *gin.Context) {
	var (
		err         error
		ctx, cancel = context.WithTimeout(c.Request.Context(), s.timeoutPerRequest)
	)
	defer func() {
		cancel()
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	body, err := s.verify(c.Request)
	if err != nil {
		err = fmt.Errorf("verifying slack event: %w", err)
		_ = c.Error(err)
		err = errUnauthorized
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(body, slackevents.OptionNoVerifyToken())
	if err != nil {
		err = fmt.Errorf("parsing event for slack: %w", err)
		_ = c.Error(err)
		err = errInvalidEvent
		return
	}

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification: // for slack webhook verification
		urlVerificationEvent, ok := eventsAPIEvent.Data.(*slackevents.EventsAPIURLVerificationEvent)
		if !ok {
			err = fmt.Errorf("invalid slack vecification event: %w", err)
			_ = c.Error(err)
			err = errInvalidWebhookCall
			return
		}
		response := slackevents.ChallengeResponse{
			Challenge: urlVerificationEvent.Challenge,
		}
		c.PureJSON(http.StatusOK, response)

	case slackevents.CallbackEvent: // real slack events
		err = s.deliverEvent(ctx, eventsAPIEvent)
		if err != nil {
			err = fmt.Errorf("delivering event: %w", err)
			_ = c.Error(err)
			err = errInvalidWebhookCall
			return
		}
	default:
		err = errInvalidEvent
	}
}

func (s *SlackWebhook) verify(req *http.Request) (body json.RawMessage, err error) {
	defer req.Body.Close()
	body, err = io.ReadAll(req.Body)
	if err != nil {
		err = fmt.Errorf("reading request body: %w", err)
		return
	}

	if s.skipSignVerification {
		return
	}

	sv, err := slack.NewSecretsVerifier(req.Header, s.signingSecret)
	if err != nil {
		err = fmt.Errorf("creating secret verifier for slack: %w", err)
		return
	}

	if _, err = sv.Write(body); err != nil {
		err = fmt.Errorf("writing secret verifier for slack: %w", err)
		return
	}

	if err = sv.Ensure(); err != nil {
		err = fmt.Errorf("failed to authorize as slack: %w", err)
		return
	}
	return
}

func (s *SlackWebhook) deliverEvent(ctx context.Context, eventsAPIEvent slackevents.EventsAPIEvent) (err error) {
	innerEvent := eventsAPIEvent.InnerEvent
	var data []byte
	data, err = json.Marshal(innerEvent.Data)
	if err != nil {
		err = fmt.Errorf("marshing inner event data: %w", err)
		return
	}

	if innerEvent.Type == slack.TYPE_MESSAGE {
		return s.deliverMessageEvent(ctx, eventsAPIEvent, data)
	}
	return
}

func (s *SlackWebhook) deliverMessageEvent(ctx context.Context, event slackevents.EventsAPIEvent, data []byte) (err error) {
	messageEvent, ok := event.InnerEvent.Data.(*slackevents.MessageEvent)
	if !ok {
		return
	}

	// if this message sent by a bot, can't trigger again.
	// TODO(sword): maybe this should be a optional.
	if messageEvent.BotID != "" {
		return
	}

	// reference: slack.MsgSubTypeBotMessage
	// document: https://api.slack.com/events/message
	if messageEvent.SubType != "" {
		return
	}

	if messageEvent.Text == "" {
		return
	}

	triggers, err := s.db.GetTriggersByQueryID(ctx, event.TeamID)
	if err != nil {
		err = fmt.Errorf("querying triggers by query id %q: %w", event.TeamID, err)
		return
	}

	for _, slackTrigger := range triggers {
		targetEvent, ok := slackTrigger.Data[TargetSlackEvent].(string)
		if !ok || targetEvent != slack.TYPE_MESSAGE {
			s.logger.Warn("delivering event: no target slack event found", log.Any("trigger", slackTrigger))
			continue
		}

		channelID, ok := slackTrigger.Data[SlackChannelID].(string)
		if !ok || channelID == "" {
			s.logger.Warn("delivering event: no target slack channelID found", log.Any("trigger", slackTrigger))
			continue
		}

		if messageEvent.Channel != channelID {
			continue
		}

		workload := work.Work{
			WorkflowID:       slackTrigger.WorkflowID,
			StartNodeID:      slackTrigger.NodeID,
			StartNodePayload: data,
		}
		err = s.producer.Produce(ctx, &workload)
		if err != nil {
			err = fmt.Errorf("pushing workload: %w", err)
			return
		}
	}
	return
}
