package schedule

import (
	_ "embed"

	"github.com/golang-module/carbon"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func init() {
	workflow.RegistryNodeMeta(&CronTrigger{})
}

// CronTrigger just define a empty struct.
type CronTrigger struct{}

var _ trigger.SampleProvider = (*CronTrigger)(nil)

type response struct {
	Datetime  string `json:"datetime"`
	IsWeekday bool   `json:"isWeekday"`
}

var _ trigger.SampleData = (*response)(nil)

func (r response) GetID() string {
	return r.Datetime
}

func (r response) GetVersion() string {
	return r.Datetime
}

func (t *CronTrigger) GetConfigObject() any {
	return &CronTrigger{}
}

func (t *CronTrigger) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	result = []trigger.SampleData{
		t.run(),
	}
	return
}

func (*CronTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("cron"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CronTrigger)
		},
		InputForm: spec.InputSchema,
	}
}

func (t *CronTrigger) Run(c *workflow.NodeContext) (any, error) {
	return t.run(), nil
}

func (*CronTrigger) run() response {
	now := carbon.Now()
	return response{
		Datetime:  now.ToDateTimeString(),
		IsWeekday: now.IsWeekday(),
	}
}
