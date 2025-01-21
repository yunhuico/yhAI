package cache

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type CacheSuite struct {
	onceRedis   sync.Once
	ca          *Cache
	purgeDocker func()
}

var Suite = &CacheSuite{}

func (c *CacheSuite) Run(t *testing.T, test func(t *testing.T, ctx context.Context, c *Cache)) {
	t.Helper()

	if os.Getenv("TEST_REDIS") != "1" {
		t.SkipNow()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	test(t, ctx, c.RedisCache())
}

func (c *CacheSuite) RedisCache() *Cache {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.onceRedis.Do(func() {
		var (
			err  error
			addr = os.Getenv("REDIS_ADDR")
		)
		defer func() {
			if err != nil {
				panic(err)
			}
		}()

		if addr != "" {
			c.ca, err = NewCache(ctx, RedisConfig{
				Addr: addr,
				DB:   0,
			})
			if err != nil {
				err = fmt.Errorf("NewCache Redis: %w", err)
				return
			}
			return
		}

		var pool *dockertest.Pool
		pool, err = dockertest.NewPool("")
		if err != nil {
			err = fmt.Errorf("could not connect to docker: %w", err)
			return
		}
		// pulls an image, creates a container based on it and runs it
		var redisResource *dockertest.Resource
		redisResource, err = pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "redis",
			Tag:        "6",
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		})
		if err != nil {
			err = fmt.Errorf("could not start redis: %w", err)
			return
		}
		err = redisResource.Expire(2 * 60)
		if err != nil {
			err = fmt.Errorf("[resource leaking] failed to set container expire: %w", err)
			return
		}
		c.purgeDocker = func() {
			err := pool.Purge(redisResource)
			if err != nil {
				err = fmt.Errorf("purging redis: %w", err)
				log.Printf("resource leaked: %s", err)
			}
		}

		// determine the port the container is listening on
		addr = net.JoinHostPort("localhost", redisResource.GetPort("6379/tcp"))
		// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
		err = pool.Retry(func() (err error) {
			c.ca, err = NewCache(ctx, RedisConfig{
				Addr: addr,
				DB:   0,
			})

			log.Printf("connecting to Redis, err: %s", err)
			return
		})
		if err != nil {
			err = fmt.Errorf("could not connect to redis container: %w", err)
			return
		}

	})

	err := c.ca.FlushAll(ctx)
	if err != nil {
		err = fmt.Errorf("flusing Redis: %w", err)
		panic(err)
	}

	return c.ca
}
