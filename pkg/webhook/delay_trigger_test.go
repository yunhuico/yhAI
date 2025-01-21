package webhook

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

const (
	KEY  = "object_key"
	DATA = "object_data"
)

type mockCache struct {
	ids  map[string][]string
	data map[string]string
}

type mockProducer struct {
	work *work.Work
	done chan bool
}

func (m *mockProducer) Produce(ctx context.Context, work *work.Work) (err error) {
	m.work = work
	m.done <- true
	return nil
}

func (m *mockProducer) GetWork() (*work.Work, error) {
	<-m.done
	if m.work == nil {
		return nil, errors.New("empty")
	}
	return m.work, nil
}

func (m *mockCache) CreateSObjectData(ctx context.Context, score float64, sobjectID string, data []byte) error {
	m.ids[KEY] = append(m.ids[KEY], sobjectID)
	m.data[DATA] = string(data)
	return nil
}

func (m *mockCache) GetSObjectIDs(ctx context.Context, min, max string, limit int) ([]string, error) {
	if v, ok := m.ids[KEY]; ok {
		return v, nil
	}
	return nil, errors.New("empty")
}

func (m *mockCache) PopSObjectData(ctx context.Context, sobjectID string) ([]byte, error) {
	if v, ok := m.data[DATA]; ok {
		return []byte(v), nil
	}
	return nil, errors.New("empty")
}

func TestDelayTrigger(t *testing.T) {
	ctx := context.Background()
	ca := &mockCache{
		ids:  map[string][]string{},
		data: map[string]string{},
	}
	p := &mockProducer{
		done: make(chan bool),
	}
	delay := NewDelayTrigger(ctx, ca, p)
	assert.NotNil(t, delay)

	req := &workflow.HTTPRequest{
		Header: map[string]string{
			"TEST-01": "01-data",
			"TEST-02": "02-data",
		},
		Body: []byte(`{"x":"Hello world"}`),
	}

	err := delay.Push(ctx, "triggerID:webhookID:workflowID:nodeID", req)
	assert.NoError(t, err)

	delay.Start()
	nw, err := p.GetWork()
	assert.NoError(t, err)
	assert.Equal(t, nw.WorkflowID, "workflowID")
	assert.Equal(t, nw.StartNodeID, "nodeID")

	delay.Stop()
	assert.False(t, strings.Contains(delay.min, "("))
	assert.True(t, strings.Contains(delay.max, "("))
}
