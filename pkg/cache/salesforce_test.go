package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSalesforceCache(t *testing.T) {
	Suite.Run(t, func(t *testing.T, ctx context.Context, ca *Cache) {
		err := ca.CreateSObjectData(ctx, 12, "sobject-01", []byte(`{"x":"hello"}`))
		assert.NoError(t, err)
		err = ca.CreateSObjectData(ctx, 17, "sobject-02", []byte(`{"x":"world"}`))
		assert.NoError(t, err)

		err = ca.CreateSObjectData(ctx, 12, "sobject-03", []byte(`{"x":"hello"}`))
		assert.NoError(t, err)

		res, err := ca.GetSObjectIDs(ctx, "0", "12", 2)
		assert.NoError(t, err)
		assert.Equal(t, len(res), 2)

		res, err = ca.GetSObjectIDs(ctx, "0", "12", 2)
		assert.NoError(t, err)
		assert.Equal(t, len(res), 0)

		res, err = ca.GetSObjectIDs(ctx, "0", "17", 1)
		assert.NoError(t, err)
		assert.Equal(t, len(res), 1)

		res, err = ca.GetSObjectIDs(ctx, "0", "17", 5)
		assert.NoError(t, err)
		assert.Equal(t, len(res), 0)

		val, err := ca.PopSObjectData(ctx, "sobject-01")
		assert.NotNil(t, val)
		assert.NoError(t, err)

		val, err = ca.PopSObjectData(ctx, "sobject-01")
		assert.Nil(t, val)
		assert.Error(t, err)
	})
}
