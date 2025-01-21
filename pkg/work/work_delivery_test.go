package work

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func mustNanoID() string {
	id, err := utils.NanoID()
	if err != nil {
		panic(err)
	}

	return id
}

// createTopic creates one or more topics and makes sure they are all ready for use.
// The caller MUST guarantee that the topic names do not collide between tests,
// using nanoID is a good trick.
func createTopic(ctx context.Context, client *kafka.Client, topic ...string) (err error) {
	topicConfigs := make([]kafka.TopicConfig, 0, len(topic)+1)
	for _, item := range topic {
		topicConfigs = append(topicConfigs, kafka.TopicConfig{
			Topic:             item,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	canaryTopicName, err := utils.NanoID()
	if err != nil {
		err = fmt.Errorf("generatin nanoID: %w", err)
		return
	}
	topicConfigs = append(topicConfigs, kafka.TopicConfig{
		Topic:             canaryTopicName,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})

	resp, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
		Topics: topicConfigs,
	})
	if err != nil {
		err = fmt.Errorf("creating topics: %w", err)
		return
	}
	for k, v := range resp.Errors {
		if v != nil {
			err = fmt.Errorf("creating topic error on %q: %w", k, v)
			return
		}
	}

	// when canary topic is ready for writing, all topics should be up.
	writer := kafka.Writer{
		Addr:         client.Addr,
		Topic:        canaryTopicName,
		RequiredAcks: kafka.RequireOne,
	}
	defer writer.Close()

	for {
		select {
		case <-ctx.Done():
			err = fmt.Errorf("writing on canary topic: %s, the final writing error: %w", ctx.Err(), err)
			return
		default:
			// relax
		}

		err = writer.WriteMessages(ctx, kafka.Message{Value: []byte("hello world!")})
		if err == nil {
			// topics are ready
			return
		}
	}
}

// Test_WorkDelivery This test takes ~25 seconds.
func Test_WorkDelivery(t *testing.T) {
	Suite.Run(t, func(t *testing.T, ctx context.Context, addresses []string, client *kafka.Client) {
		var (
			topic = "topic-" + mustNanoID()
			group = "group-" + mustNanoID()
		)

		works := []*Work{
			{
				ID:               "", // will be overwritten
				WorkflowID:       "w1",
				StartNodeID:      "n1",
				StartNodePayload: []byte("1"),
			},
			{
				ID:               "", // will be overwritten
				WorkflowID:       "w2",
				StartNodeID:      "n2",
				StartNodePayload: []byte("2"),
			},
			{
				ID:               "", // will be overwritten
				WorkflowID:       "w3",
				StartNodeID:      "n3",
				StartNodePayload: []byte("3"),
			},
			{
				ID:               "instance-4", // a paused work
				WorkflowID:       "w4",
				StartNodeID:      "n4",
				StartNodePayload: []byte("4"),
				Resume:           true,
			},
		}
		store := new(dummyStore)
		store.M = make(map[string]*model.WorkflowInstance)
		store.M["instance-4"] = &model.WorkflowInstance{
			ID:         "instance-4",
			WorkflowID: "w4",
			Status:     model.WorkflowInstanceStatusPaused,
		}

		assert := require.New(t)

		err := createTopic(ctx, client, topic)
		assert.NoError(err)

		producer, err := NewProducer(ctx, ProducerOpt{
			DB: store,
			ProducerConfig: ProducerConfig{
				KafkaTopic:     topic,
				KafkaAddresses: addresses,
			},
		})
		assert.NoError(err)
		defer producer.Close()

		for _, work := range works {
			err = producer.Produce(ctx, work)
			assert.NoError(err)
			require.NotEmpty(t, work.ID)
			require.NotNil(t, store.M[work.ID])
			require.Equal(t, store.M[work.ID].Status, model.WorkflowInstanceStatusScheduled)
		}

		consumer, err := NewConsumer(ctx, ConsumerOpt{
			DB: store,
			ConsumerConfig: ConsumerConfig{
				KafkaTopic:     topic,
				KafkaAddresses: addresses,
				ConsumerGroup:  group,
			},
		})
		require.NoError(t, err)
		defer consumer.Close()

		var gotWork Work
		for _, wantWork := range works {
			gotWork, err = consumer.Consume(ctx)
			require.NoError(t, err)
			require.Equal(t, wantWork, &gotWork)
			require.Equal(t, store.M[gotWork.ID].Status, model.WorkflowInstanceStatusRunning)
		}
	})
}

func mockWork(seq int) *Work {
	id := strconv.Itoa(seq)

	return &Work{
		ID:               "", // will be overwritten
		WorkflowID:       "w" + id,
		StartNodeID:      "n" + id,
		StartNodePayload: []byte(id),
	}
}

// Test_WorkDeliveryConcurrently This test takes ~25 seconds.
func Test_WorkDeliveryConcurrently(t *testing.T) {
	Suite.Run(t, func(t *testing.T, ctx context.Context, addresses []string, client *kafka.Client) {
		var (
			topic = "topic-" + mustNanoID()
			group = "group-" + mustNanoID()
		)

		store := new(dummyStore)

		assert := require.New(t)

		err := createTopic(ctx, client, topic)
		assert.NoError(err)

		producer, err := NewProducer(ctx, ProducerOpt{
			DB: store,
			ProducerConfig: ProducerConfig{
				KafkaTopic:     topic,
				KafkaAddresses: addresses,
			},
		})
		assert.NoError(err)
		defer producer.Close()

		var wg sync.WaitGroup

		produceWork := func(from, to int) {
			defer wg.Done()
			for i := from; i < to; i++ {
				// uncomment me if there's a problem:
				// t.Log("producing work: ", i)
				work := mockWork(i)

				var produceStartedAt = time.Now()
				bad := producer.Produce(ctx, work)
				assert.NoError(bad)
				assert.NotEmpty(work.ID)
				t.Log("work ", i, " produced, time taken ", time.Since(produceStartedAt).String())
			}
		}

		startedAt := time.Now()

		const max = 50
		wg.Add(5)
		go produceWork(0, 10)
		go produceWork(10, 20)
		go produceWork(20, 30)
		go produceWork(30, 40)
		go produceWork(40, max)
		wg.Wait()

		t.Log("producing into kafka time taken: ", time.Since(startedAt).String())

		consumer, err := NewConsumer(ctx, ConsumerOpt{
			DB: store,
			ConsumerConfig: ConsumerConfig{
				KafkaTopic:     topic,
				KafkaAddresses: addresses,
				ConsumerGroup:  group,
			},
		})
		require.NoError(t, err)
		defer consumer.Close()

		good := make(chan struct{})
		go func() {
			var (
				i              int
				drainStartedAt = time.Now()
			)
			for {
				gotWork, err := consumer.Consume(ctx)
				require.NoError(t, err)
				require.Equal(t, store.M[gotWork.ID].Status, model.WorkflowInstanceStatusRunning)
				i++
				if i >= max {
					close(good)
					t.Log("drain topic time taken: ", time.Since(drainStartedAt).String())
					return
				}
			}
		}()

		select {
		case <-ctx.Done():
			assert.NoError(ctx.Err())
		case <-good:
			return
		}
	})
}

type dummyStore struct {
	mu sync.Mutex
	M  map[string]*model.WorkflowInstance
}

func (d *dummyStore) UpdateWorkflowInstanceStatusByID(ctx context.Context, id string, newStatus model.WorkflowInstanceStatus, wantStatus model.WorkflowInstanceStatus) (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if id == "" {
		err = errors.New("provided id is empty")
		return
	}

	if d.M == nil {
		d.M = make(map[string]*model.WorkflowInstance)
	}

	instance := d.M[id]
	if instance == nil {
		err = fmt.Errorf("instance %q not existed", id)
		return
	}
	if wantStatus != "" && instance.Status != wantStatus {
		err = fmt.Errorf("instance %q status is %q, want %q", id, instance.Status, wantStatus)
		return
	}

	instance.Status = newStatus
	return
}

func (d *dummyStore) MarkWorkflowInstanceAsRunning(ctx context.Context, id string) (ok bool, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if id == "" {
		err = errors.New("provided id is empty")
		return
	}

	if d.M == nil {
		d.M = make(map[string]*model.WorkflowInstance)
	}

	instance := d.M[id]
	if instance == nil {
		// relax
		return
	}
	if instance.Status != model.WorkflowInstanceStatusScheduled {
		// relax
		return
	}

	instance.Status = model.WorkflowInstanceStatusRunning
	ok = true

	return
}

func (d *dummyStore) InsertWorkflowInstance(ctx context.Context, instance *model.WorkflowInstance) (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if instance.ID == "" {
		err = errors.New("instance id is empty")
		return
	}

	if d.M == nil {
		d.M = make(map[string]*model.WorkflowInstance)
	}

	d.M[instance.ID] = instance

	return
}
