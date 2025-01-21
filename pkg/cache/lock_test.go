package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/atomic"

	"github.com/go-redis/redis/v9"

	"github.com/stretchr/testify/require"
)

func TestCache_acquireLock(t *testing.T) {
	Suite.Run(t, func(t *testing.T, ctx context.Context, c *Cache) {
		var (
			err       error
			randomStr string
			assert    = require.New(t)
		)

		_, err = c.acquireLock(ctx, "", time.Second, false)
		assert.Error(err)

		randomStr, err = c.acquireLock(ctx, "hello", 5*time.Second, false)
		assert.NoError(err)
		assert.NotEmpty(randomStr)
		gotStr, err := c.core.Get(ctx, lockKeyPrefix+"hello").Result()
		assert.NoError(err)
		assert.Equal(randomStr, gotStr)

		_, err = c.acquireLock(ctx, "hello", 5*time.Second, false)
		assert.Error(err)
		assert.Equal(ErrResourceAlreadyLocked, err)

		err = c.releaseLock(ctx, "hello", "not me")
		assert.Error(err)
		assert.Equal(ErrResourceConflict, err)

		err = c.releaseLock(ctx, "hello", randomStr)
		assert.NoError(err)
		_, err = c.core.Get(ctx, lockKeyPrefix+"hello").Result()
		assert.Error(err)
		assert.Equal(redis.Nil, err)

		err = c.releaseLock(ctx, "hello", randomStr)
		assert.Error(err)
		assert.Equal(ErrResourceNotLocked, err)
	})
}

func TestCache_WaitLockToRun(t *testing.T) {
	Suite.Run(t, func(t *testing.T, ctx context.Context, c *Cache) {
		var (
			err         error
			assert      = require.New(t)
			expectedErr = errors.New("expected error")
		)

		err = c.WaitLockToRun(ctx, "name-1", 1*time.Second, func() error {
			return nil
		})
		assert.NoError(err)

		err = c.WaitLockToRun(ctx, "name-0", 1*time.Second, func() error {
			return expectedErr
		})
		assert.True(errors.Is(err, expectedErr))

		var (
			barrier = make(chan struct{})
			signal  atomic.Bool
		)
		go func() {
			err = c.WaitLockToRun(ctx, "name-2", 10*time.Second, func() error {
				close(barrier)
				time.Sleep(200 * time.Millisecond)
				signal.Store(true)
				return nil
			})
			assert.NoError(err)
		}()

		<-barrier

		err = c.WaitLockToRun(ctx, "name-2", 10*time.Second, func() error {
			assert.True(signal.Load())
			return nil
		})
		assert.NoError(err)

		// time out situation
		barrier = make(chan struct{})

		go func() {
			err = c.WaitLockToRun(ctx, "name-3", 10*time.Second, func() error {
				close(barrier)
				time.Sleep(500 * time.Millisecond)
				return nil
			})
			assert.NoError(err)
		}()

		<-barrier

		waitingCtx, cancelWait := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancelWait()
		err = c.WaitLockToRun(waitingCtx, "name-3", 10*time.Second, func() error {
			return nil
		})
		assert.Error(err)
		assert.True(errors.Is(err, context.DeadlineExceeded))
	})
}
