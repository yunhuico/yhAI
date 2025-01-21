package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v9"
)

//go:embed create_sobject.lua
var createSObjectScript string

//go:embed get_expired_sobject.lua
var getExpiredSObjectScript string

//go:embed pop_sobject.lua
var popSObjectScript string

var (
	salesforceCreateScript = redis.NewScript(createSObjectScript)
	salesforceGetScript    = redis.NewScript(getExpiredSObjectScript)
	salesforcePopScript    = redis.NewScript(popSObjectScript)
)

const (
	SOBJECTKEY  = "salesforce_objects"
	SOBJECTDATA = "salesforce_object_data"
)

// CreateSObjectData create a zset value: id->sobjectID, score->score, key->keys[0],
// and a hset: key-> keys[1], field->sobjectID, value->data
func (c *Cache) CreateSObjectData(ctx context.Context, score float64, sobjectID string, data []byte) error {
	err := salesforceCreateScript.Run(ctx, c.core, []string{SOBJECTKEY, SOBJECTDATA}, sobjectID, score, data).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}
	return nil
}

// GetSObjectIDs gets all ids with score in [min,max)
func (c *Cache) GetSObjectIDs(ctx context.Context, min, max string, limit int) ([]string, error) {
	val, err := salesforceGetScript.Run(ctx, c.core, []string{SOBJECTKEY}, min, max, limit).StringSlice()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return val, nil
		}
		return nil, fmt.Errorf("get sobject id: %w", err)
	}
	return val, nil
}

// PopSObjectData pop a sobject data from hset
func (c *Cache) PopSObjectData(ctx context.Context, sobjectID string) ([]byte, error) {
	val := salesforcePopScript.Run(ctx, c.core, []string{SOBJECTDATA}, sobjectID).Val()
	if val == nil {
		return nil, errors.New("pop sobject data error")
	}
	if v, ok := val.(string); ok {
		return []byte(v), nil
	}
	return nil, errors.New("invalid type")
}
