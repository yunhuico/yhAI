package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

const (
	defaultExpiredTime     = 10
	defaultTriggerDuration = 1
)

type SObjectOperator interface {
	CreateSObjectData(ctx context.Context, score float64, sobjectID string, data []byte) error
	GetSObjectIDs(ctx context.Context, min, max string, limit int) ([]string, error)
	PopSObjectData(ctx context.Context, sobjectID string) ([]byte, error)
}

type DelayTrigger struct {
	store    SObjectOperator
	ctx      context.Context
	producer WorkProducer
	min      string
	max      string
	logger   log.Logger
	done     chan bool
}

func NewDelayTrigger(ctx context.Context, store SObjectOperator, producer WorkProducer) *DelayTrigger {
	return &DelayTrigger{
		ctx:      ctx,
		store:    store,
		producer: producer,
		min:      "0",
		max:      fmt.Sprintf("(%d", time.Now().UnixMilli()),
		done:     make(chan bool),
		logger:   log.Clone(log.Namespace("trigger/delayTrigger")),
	}
}

func (t *DelayTrigger) Push(ctx context.Context, triggerID string, httpRequest *workflow.HTTPRequest) error {
	data, err := json.Marshal(httpRequest)
	if err != nil {
		return err
	}
	expired := time.Now().Add(time.Second * defaultExpiredTime).UnixMilli()
	err = t.store.CreateSObjectData(ctx, float64(expired), triggerID, data)
	return err
}

func (t *DelayTrigger) run() {
	for {
		ticker := time.NewTicker(time.Second * defaultTriggerDuration)
		select {
		case <-ticker.C:
			t.checkAndTriggerWorkflow()
		case <-t.done:
			ticker.Stop()
			return
		}
	}
}

func (t *DelayTrigger) Stop() {
	t.done <- true
}

func (t *DelayTrigger) Start() {
	go t.run()
}

func (t *DelayTrigger) checkAndTriggerWorkflow() {
	res, err := t.store.GetSObjectIDs(t.ctx, t.min, t.max, 10)
	if err != nil {
		t.logger.For(t.ctx).Warn("get sobject ids: %w", log.ErrField(err))
		return
	}
	for i := range res {
		ids := strings.Split(res[i], ":")
		if len(ids) != 4 {
			t.logger.For(t.ctx).Warn("invalid id")
			continue
		}
		data, err := t.store.PopSObjectData(t.ctx, res[i])
		if err != nil {
			t.logger.For(t.ctx).Warn("hget sobject data: %w", log.ErrField(err))
			continue
		}

		workload := work.Work{
			WorkflowID:       ids[2],
			StartNodeID:      ids[3],
			StartNodePayload: data,
		}

		err = t.producer.Produce(t.ctx, &workload)
		if err != nil {
			log.Warn("pushing workload", log.ErrField(err))
		}
	}
	t.min = strings.Trim(t.max, "(")
	t.max = fmt.Sprintf("(%d", time.Now().Add(time.Second*defaultTriggerDuration).UnixMilli())
}
