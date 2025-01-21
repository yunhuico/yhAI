package trigger

import (
	"context"
	"fmt"
	logging "log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

type DkronSuite struct {
	onceDkron   sync.Once
	purgeDocker func()
	client      *DkronClient
}

func checkDkronService(cronServerHost string) error {
	url := fmt.Sprintf("%s/v1/leader", cronServerHost)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("performing http request for server %q: %w", cronServerHost, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dkron has not started, repsonded with code %q", resp.StatusCode)
	}
	return nil
}

func (d *DkronSuite) Close() {
	if d.purgeDocker != nil {
		d.purgeDocker()
	}
}

func (d *DkronSuite) Run(t *testing.T, test func(t *testing.T, ctx context.Context, client *DkronClient)) {
	t.Helper()
	if os.Getenv("TEST_DKRON") != "1" {
		t.SkipNow()
	}

	client := d.Dkron()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	test(t, ctx, client)
}

var testJobTags = map[string]string{"dc": "dc1:1"}

func (d *DkronSuite) Dkron() (client *DkronClient) {
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	d.onceDkron.Do(func() {
		address := os.Getenv("DKRON_ADDR")

		if address != "" {
			err = checkDkronService(address)
			if err != nil {
				err = fmt.Errorf("checking dkron: %w", err)
				return
			}
			client, err = NewDkronClient(DkronConfig{
				DkronInternalHost:   address,
				WebhookInternalHost: "http://127.0.0.1:65000",
				JobTags:             testJobTags,
			})
			if err != nil {
				err = fmt.Errorf("initializing Dkron client: %w", err)
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
		var dkron *dockertest.Resource

		dkron, err = pool.RunWithOptions(&dockertest.RunOptions{
			Name:       "dkron",
			Repository: "dkron/dkron",
			Tag:        "3.2.1",
			Cmd:        []string{"agent", "--server", "--bootstrap-expect=1", "--node-name=node1"},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		})
		if err != nil {
			err = fmt.Errorf("could not start dkron: %w", err)
			return
		}
		err = dkron.Expire(2 * 60)
		if err != nil {
			err = fmt.Errorf("[resource leaking] failed to set container expire: %w", err)
			return
		}
		d.purgeDocker = func() {
			err := pool.Purge(dkron)
			if err != nil {
				err = fmt.Errorf("purging dkron: %w", err)
				logging.Printf("resource leaked: %s", err)
			}
		}

		// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
		address = fmt.Sprintf("http://localhost:%s", dkron.GetPort("8080/tcp"))

		err = pool.Retry(func() (err error) {
			err = checkDkronService(address)
			return
		})
		if err != nil {
			err = fmt.Errorf("could not connect to dkron container: %w", err)
			return
		}

		client, err = NewDkronClient(DkronConfig{
			DkronInternalHost:   address,
			WebhookInternalHost: "http://127.0.0.1:65000",
			JobTags:             map[string]string{"dc": "dc1:1"},
		})
		if err != nil {
			err = fmt.Errorf("initializing Dkron client: %w", err)
			return
		}

	})
	if err != nil {
		return
	}

	d.client = client

	return
}

func TestCronTrigger(t *testing.T) {
	var (
		assert = require.New(t)

		triggerA = model.Trigger{
			ID:           "a",
			WorkflowID:   "workflow-a",
			NodeID:       "node-a",
			Type:         model.TriggerTypeCron,
			Name:         "trigger a",
			AdapterClass: "ultrafox/trigger",
		}
		triggerB = model.Trigger{
			ID:           "b",
			WorkflowID:   "workflow-b",
			NodeID:       "node-b",
			Type:         model.TriggerTypeCron,
			Name:         "trigger b",
			AdapterClass: "ultrafox/trigger",
		}
	)

	dkronSuite.Run(t, func(t *testing.T, ctx context.Context, client *DkronClient) {
		_, err := client.UpsertJob(ctx, triggerA.ID, "@every 1s", "Asia/Tokyo", map[string]any{"a": 1})
		assert.NoError(err)
		jobs, err := client.GetJobs(ctx)
		assert.NoError(err)
		assert.Len(jobs, 1)
		assert.Equal(testJobTags, jobs[0].Tags)
		assert.Equal("Asia/Tokyo", jobs[0].Timezone)
		assert.Equal(`{"a":1}`, jobs[0].ExecutorConfig.Body)

		_, err = client.UpsertJob(ctx, triggerB.ID, "@every 5s", "", nil)
		assert.NoError(err)

		err = client.DeleteJob(ctx, triggerA.ID)
		assert.NoError(err)

		err = client.DeleteJob(ctx, triggerB.ID)
		assert.NoError(err)
	})
}
