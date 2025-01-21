package webhook

import (
	"embed"
	"encoding/json"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapter := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapter.RegisterSpecsByDir(adapterDir)

	workflow.RegistryNodeMeta(&ReceiveTrigger{})
}

var _ trigger.TriggerProvider = (*ReceiveTrigger)(nil)

type ReceiveTrigger workflow.HTTPRequest

func (t *ReceiveTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/webhook#receiveData")
	return workflow.NodeMeta{
		Class:     spec.Class,
		InputForm: spec.InputSchema,
		New: func() workflow.Node {
			return new(ReceiveTrigger)
		},
	}
}

type response struct {
	Header map[string]string `json:"header"`
	Query  map[string]string `json:"query"`
	Body   any               `json:"body"`
}

func (t *ReceiveTrigger) Run(_ *workflow.NodeContext) (result any, err error) {
	resp := response{
		Header: t.Header,
		Query:  t.Query,
	}
	if len(t.Body) == 0 {
		return resp, nil
	}
	err = json.Unmarshal(t.Body, &resp.Body)
	if err != nil {
		err = fmt.Errorf("webhook body invalid: %w", err)
		return
	}
	return resp, nil
}

func (t *ReceiveTrigger) GetConfigObject() any {
	return nil
}

func (t *ReceiveTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return nil, nil
}

func (t *ReceiveTrigger) Delete(c trigger.WebhookContext) error {
	return nil
}
