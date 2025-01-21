package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

var ErrKVKeyNotExisted = errors.New("kv: key not existed")

// KVProcessor handles workflow persistent KV operation
type KVProcessor struct {
	workflowID string
	db         *model.DB
}

// Get gets value by key.
func (k *KVProcessor) Get(ctx context.Context, key string) (value any, err error) {
	kv, err := k.db.GetKVByWorkflow(ctx, k.workflowID, key)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrKVKeyNotExisted
		return
	}
	if err != nil {
		err = fmt.Errorf("querying database: %w", err)
		return
	}
	if len(kv.Value) == 0 {
		return
	}

	value = new(any)
	err = json.Unmarshal(kv.Value, value)
	if err != nil {
		err = fmt.Errorf("unmarshaling JSON: %w", err)
		return
	}

	return
}

// Set sets the value by the key.
func (k *KVProcessor) Set(ctx context.Context, key string, value any) (err error) {
	err = k.db.SetKVByWorkflow(ctx, k.workflowID, key, value)
	if err != nil {
		err = fmt.Errorf("KVDao: %w", err)
		return
	}

	return
}
