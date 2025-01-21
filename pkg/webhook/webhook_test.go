package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	_ "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/schedule"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

func TestWebhookTrigger_handleWebhook(t *testing.T) {
	var (
		err    error
		assert = require.New(t)
	)

	logger, err := log.New("gotest:webhook", log.DebugLevel)
	assert.NoError(err)

	producer := &dummyWebhookWorkProducer{}
	webhookTrigger, err := NewWebhookTrigger(WebhookTriggerOpt{
		DB:       dummyWebhookTriggerStore{},
		Logger:   logger,
		Producer: producer,
	}, SlackWebhookOpt{SigningSecret: "abcdefg12345"}, SalesforceWebhookOpt{}, DingtalkCorpBotWebhookOpt{})
	assert.NoError(err)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://example.com/hooks/a?hello=world", nil)
	webhookTrigger.router.ServeHTTP(recorder, req)
	resp := recorder.Result()

	assert.Equal(http.StatusOK, resp.StatusCode)
	assert.NoError(bodyIsOK(resp.Body))
	produced := producer.Produced
	producer.Clear()
	assert.NotEmpty(produced)
	assert.Equal(produced.WorkflowID, triggerA.WorkflowID)
	assert.Equal(produced.StartNodeID, triggerA.NodeID)
	first := &workflow.HTTPRequest{
		Header: map[string]string{},
		Query: map[string]string{
			"hello": "world",
		},
		Body: []byte{},
	}
	data, err := first.Marshal()
	assert.NoError(err)
	assert.Equal(produced.StartNodePayload, data)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "https://example.com/hooks/b", bytes.NewReader([]byte("hello world!")))
	req.Header.Set("X-Foo", "Bar")
	webhookTrigger.router.ServeHTTP(recorder, req)
	resp = recorder.Result()

	assert.Equal(http.StatusOK, resp.StatusCode)
	assert.NoError(bodyIsOK(resp.Body))
	produced = producer.Produced
	producer.Clear()
	assert.NotEmpty(produced)
	assert.Equal(produced.WorkflowID, triggerB.WorkflowID)
	assert.Equal(produced.StartNodeID, triggerB.NodeID)
	second := &workflow.HTTPRequest{
		Header: map[string]string{
			"X-Foo": "Bar",
		},
		Query: map[string]string{},
		Body:  []byte("hello world!"),
	}
	data, err = second.Marshal()
	assert.NoError(err)
	assert.Equal(produced.StartNodePayload, data)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "https://example.com/hooks/not-existed", bytes.NewReader([]byte("hello world!")))
	webhookTrigger.router.ServeHTTP(recorder, req)
	resp = recorder.Result()

	assert.Equal(http.StatusNotAcceptable, resp.StatusCode)
	assert.Error(bodyIsOK(resp.Body))
	assert.Nil(producer.Produced)
}

func bodyIsOK(r io.Reader) (err error) {
	decoder := json.NewDecoder(r)

	resp := httpbase.R{Code: -1}
	err = decoder.Decode(&resp)
	if err != nil {
		err = fmt.Errorf("decodig JSON into R: %w", err)
		return
	}
	if resp.Code != 0 {
		err = fmt.Errorf("resp code %d, msg: %s", resp.Code, resp.Msg)
		return
	}

	return
}

var (
	triggerA = model.TriggerWithNode{
		Trigger: model.Trigger{
			ID:           "a",
			WorkflowID:   "workflow-a",
			NodeID:       "node-a",
			Type:         model.TriggerTypeWebhook,
			Name:         "trigger a",
			AdapterClass: "ultrafox/a",
		},
		Node: &model.Node{
			EditableNode: model.EditableNode{
				Class: validate.CronTriggerClass,
			},
		},
	}
	triggerB = model.TriggerWithNode{
		Trigger: model.Trigger{
			ID:           "b",
			WorkflowID:   "workflow-b",
			NodeID:       "node-b",
			Type:         model.TriggerTypeWebhook,
			Name:         "trigger b",
			AdapterClass: "ultrafox/b",
		},
		Node: &model.Node{
			EditableNode: model.EditableNode{
				Class: validate.CronTriggerClass,
			},
		},
	}
)

type dummyWebhookTriggerStore struct{}

func (d dummyWebhookTriggerStore) GetTriggerWithNodeByID(ctx context.Context, id string) (trigger model.TriggerWithNode, err error) {
	if id == "" {
		err = errors.New("id is empty")
		return
	}

	switch id {
	case "a":
		trigger = triggerA
	case "b":
		trigger = triggerB
	default:
		err = fmt.Errorf("unexpected id %q", id)
	}

	return
}

type dummyWebhookWorkProducer struct {
	Produced *work.Work
}

func (d *dummyWebhookWorkProducer) Clear() {
	d.Produced = nil
}

func (d *dummyWebhookWorkProducer) Produce(ctx context.Context, work *work.Work) (err error) {
	if work == nil {
		err = errors.New("work is nil")
		return
	}

	work.ID, err = utils.LongNanoID()
	if err != nil {
		err = fmt.Errorf("generating nanoID: %w", err)
		return
	}
	d.Produced = work

	return
}
