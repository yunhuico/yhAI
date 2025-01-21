package work

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	db     WorkflowRunningMarker
	reader *kafka.Reader
}

type WorkflowRunningMarker interface {
	MarkWorkflowInstanceAsRunning(ctx context.Context, id string) (ok bool, err error)
}

type ConsumerConfig struct {
	// which topic does the work come from
	KafkaTopic string `comment:"which topic does the work come from"`
	// kafka addresses, for example: localhost:9092
	KafkaAddresses []string `comment:"kafka addresses, for example: localhost:9092"`
	// optional, SASL authorization username and password
	SASL SASLCredential `comment:"optional, SASL authorization username and password"`
	// ID of the consumer group
	ConsumerGroup string `comment:"ID of the consumer group"`
}

type ConsumerOpt struct {
	DB WorkflowRunningMarker

	ConsumerConfig
}

// NewConsumer constructs a Consumer to fetch and handle message from Kafka.
// The caller MUST remember to close the consumer via Consumer.Close.
func NewConsumer(ctx context.Context, opt ConsumerOpt) (consumer *Consumer, err error) {
	if opt.ConsumerGroup == "" {
		err = errors.New("consumerGroup is missing")
		return
	}
	if opt.KafkaTopic == "" {
		err = errors.New("kafka topic is missing")
		return
	}

	err = pingKafka(ctx, opt.KafkaAddresses, opt.SASL)
	if err != nil {
		err = fmt.Errorf("pinging kafka: %w", err)
		return
	}

	cfg := kafka.ReaderConfig{
		Brokers:               opt.KafkaAddresses,
		GroupID:               opt.ConsumerGroup,
		Topic:                 opt.KafkaTopic,
		WatchPartitionChanges: true,
		StartOffset:           kafka.FirstOffset,
	}

	if !opt.SASL.IsEmpty() {
		cfg.Dialer = &kafka.Dialer{
			// copied from kafka.DefaultDialer
			Timeout:   10 * time.Second,
			DualStack: true,

			// https://github.com/segmentio/kafka-go/issues/799#issuecomment-979329438
			TLS: &tls.Config{},

			SASLMechanism: plain.Mechanism{
				Username: opt.SASL.Username,
				Password: opt.SASL.Password,
			},
		}
	}

	reader := kafka.NewReader(cfg)

	consumer = &Consumer{
		db:     opt.DB,
		reader: reader,
	}

	return
}

// Close closes the stream,
// preventing the program from reading any more messages from it.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// Stats returns a snapshot of the reader stats since the last time the method was called,
// or since the reader was created if it is called for the first time.
// A typical use of this method is to spawn a goroutine
// that will periodically call Stats on a kafka reader and report the metrics to a stats collection system.
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// Consume reads the next work from Kafka,
// making sure the lock is acquired.
//
// The method call blocks until a message becomes available, or an error occurs.
// The program may also specify a context to asynchronously cancel the blocking operation.
// The method returns io.EOF to indicate that the reader has been closed.
func (c *Consumer) Consume(ctx context.Context) (work Work, err error) {
FETCH:
	// is there a valid work?
	var ok bool

	message, err := c.reader.FetchMessage(ctx)
	if err == io.EOF {
		return
	}
	if err != nil {
		err = fmt.Errorf("fetching message from Kafka: %w", err)
		return
	}

	err = json.Unmarshal(message.Value, &work)
	if err != nil {
		err = fmt.Errorf("unmarshaling message into Work struct at partition %d offset %d: %w", message.Partition, message.Offset, err)
		// carry on, preventing consuming deadlock
		_ = c.reader.CommitMessages(ctx, message)
		return
	}
	err = work.validate()
	if err != nil {
		err = fmt.Errorf("validating work at partition %d offset %d: %w", message.Partition, message.Offset, err)
		// carry on, preventing consuming deadlock
		_ = c.reader.CommitMessages(ctx, message)
		return
	}

	ok, err = c.db.MarkWorkflowInstanceAsRunning(ctx, work.ID)
	if err != nil {
		err = fmt.Errorf("acquiring running lock for work: %w", err)
		return
	}

	// always commit the offset after a successful lock acquiring action:
	// - if the lock is acquired, this makes sense;
	// - if the lock is not acquired:
	//   - if the work is delivered to more than one worker, the other worker commits nevertheless and over-commit here does no harm
	//   - if the work is failed/hanging, commit here avoids workers trying acquiring an impossible-to-get lock time after time
	err = c.reader.CommitMessages(ctx, message)
	if err != nil {
		err = fmt.Errorf("commiting offset: %w", err)
		return
	}

	if !ok {
		// nothing to execute, relax
		goto FETCH
	}

	// lock acquired
	return
}
