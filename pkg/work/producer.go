package work

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/segmentio/kafka-go"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
)

// Producer is the intake of Work.
type Producer struct {
	writer *kafka.Writer
	db     WorkflowInstanceInserter
}

type WorkflowInstanceInserter interface {
	InsertWorkflowInstance(ctx context.Context, instance *model.WorkflowInstance) (err error)
	UpdateWorkflowInstanceStatusByID(ctx context.Context, id string, newStatus model.WorkflowInstanceStatus, wantStatus model.WorkflowInstanceStatus) (err error)
}

type ProducerConfig struct {
	// which topic should the producer push work into
	KafkaTopic string `comment:"which topic should the producer push work into"`
	// kafka addresses, for example: localhost:9092
	KafkaAddresses []string `comment:"kafka addresses, for example: localhost:9092"`
	// optional, SASL authorization username and password
	SASL SASLCredential `comment:"optional, SASL authorization username and password"`
	// optional, Kafka message compression, choose one: none(or an empty string), gzip, lz4, snappy, zstd.
	// zstd works best, but is only available after Kafka version 2.1.0.
	// use snappy when zstd is not available.
	// Azure Kafka does not support compression at all.
	Compression string `comment:"optional, Kafka message compression, choose one: none(or an empty string), gzip, lz4, snappy, zstd. zstd works best, but is only available after Kafka version 2.1.0. Use snappy when zstd is not available. Azure Kafka does not support compression at all."`
}

type ProducerOpt struct {
	// database instance, normally model.DB
	DB WorkflowInstanceInserter

	ProducerConfig
}

// NewProducer constructs a Producer.
// The caller MUST remember to close the producer via Producer.Close.
func NewProducer(ctx context.Context, opt ProducerOpt) (p *Producer, err error) {
	if opt.KafkaTopic == "" {
		err = errors.New("kafka topic is missing")
		return
	}

	err = pingKafka(ctx, opt.KafkaAddresses, opt.SASL)
	if err != nil {
		err = fmt.Errorf("pinging Kafka: %w", err)
		return
	}

	var compression kafka.Compression
	switch strings.ToLower(opt.Compression) {
	case "", "none":
		// relax
		//
		// Azure Kafka does not support compression at all.
		// https://learn.microsoft.com/en-us/azure/event-hubs/apache-kafka-troubleshooting-guide#compressionmessage-format-version-issue
	case "gzip":
		compression = kafka.Gzip
	case "lz4":
		compression = kafka.Lz4
	case "snappy":
		// Use Snappy when zstd is not available
		// https://blog.cloudflare.com/squeezing-the-firehose/
		compression = kafka.Snappy
	case "zstd":
		// zstd seems working better, but only available after version 2.1.0
		// https://blog.cloudflare.com/squeezing-the-firehose/
		// https://issues.apache.org/jira/browse/KAFKA-4514
		compression = kafka.Zstd
	default:
		err = fmt.Errorf("unexpected compression method, got %q", opt.Compression)
		return
	}

	writer := kafka.Writer{
		Addr:                   kafka.TCP(opt.KafkaAddresses...),
		Topic:                  opt.KafkaTopic,
		Balancer:               &kafka.RoundRobin{},
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           100 * time.Millisecond,
		MaxAttempts:            3,
		Compression:            compression,
		AllowAutoTopicCreation: false,
	}

	// ref: https://github.com/Azure/azure-event-hubs-for-kafka/blob/master/quickstart/go/producer/producer.go
	if !opt.SASL.IsEmpty() {
		writer.Transport = &kafka.Transport{
			// copied from kafka.DefaultTransport
			Dial: (&net.Dialer{
				Timeout:   3 * time.Second,
				DualStack: true,
			}).DialContext,
			// https://github.com/segmentio/kafka-go/issues/799#issuecomment-979329438
			TLS: &tls.Config{},

			SASL: plain.Mechanism{
				Username: opt.SASL.Username,
				Password: opt.SASL.Password,
			},
		}
	}

	p = &Producer{
		writer: &writer,
		db:     opt.DB,
	}

	return
}

type SASLCredential struct {
	// for Azure Event Hubs, the username is always $ConnectionString
	Username string `comment:"for Azure Event Hubs, the username is always $ConnectionString"`
	// for Azure Event Hubs, it's something like
	// Endpoint=sb://mynamespace.servicebus.windows.net/;SharedAccessKeyName=XXXXXX;SharedAccessKey=XXXXXX
	Password string `comment:"for Azure Event Hubs, it's something like Endpoint=sb://mynamespace.servicebus.windows.net/;SharedAccessKeyName=XXXXXX;SharedAccessKey=XXXXXX'"`
}

func (c SASLCredential) IsEmpty() bool {
	return c.Username == "" && c.Password == ""
}

// pingKafka checks connectivity to Kafka by request Kafka metadata.
// When there's no SASL credential needed, provide an empty one.
//
// If you bumped into error like "unexpected EOF" or "EOF" and got no clue,
// it's usually the authorization or SSL config to blame.
// (Yes, the error message is misleading)
// Refs:
// * https://github.com/segmentio/kafka-go/issues/799
// * https://stackoverflow.com/a/54302847
func pingKafka(ctx context.Context, addresses []string, saslCredential SASLCredential) (err error) {
	if len(addresses) == 0 {
		err = errors.New("no Kafka address provided")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var client kafka.Client

	// some Kafka instance, like Azure's, uses SASL authorization.
	// ref: https://github.com/Azure/azure-event-hubs-for-kafka/blob/master/quickstart/go/producer/producer.go
	if !saslCredential.IsEmpty() {
		client.Transport = &kafka.Transport{
			// copied from kafka.DefaultTransport
			Dial: (&net.Dialer{
				Timeout:   3 * time.Second,
				DualStack: true,
			}).DialContext,

			// https://github.com/segmentio/kafka-go/issues/799#issuecomment-979329438
			TLS: &tls.Config{},

			SASL: plain.Mechanism{
				Username: saslCredential.Username,
				Password: saslCredential.Password,
			},
		}
	}

	_, err = client.Metadata(ctx, &kafka.MetadataRequest{
		Addr: kafka.TCP(addresses...),
	})
	if err != nil {
		err = fmt.Errorf("fetching metadata from Kafka: %w", err)
		return
	}

	return
}

// Close flushes pending writes, and waits for all writes to complete before returning.
// Calling Close also prevents new writes from being submitted to the writer,
// further calls to Produce will fail with io.ErrClosedPipe.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Stats returns a snapshot of the writer stats since the last time the method was called,
// or since the writer was created if it is called for the first time.
// A typical use of this method is to spawn a goroutine
// that will periodically call Stats on a kafka writer and report the metrics to a stats collection system.
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}

// Produce logs the work in workflow_instances
// and pushes the work into Kafka.
//
// For brand-new work, leave the id empty and resume to false so that a new one is generated.
// To resume paused work, set resume to true and the id must be corresponding with the existing record of workflow_instances.
//
// Produce is safe for concurrent use.
func (p *Producer) Produce(ctx context.Context, work *Work) (err error) {
	if work == nil {
		err = errors.New("provided work is nil")
		return
	}

	if !work.Resume {
		work.ID, err = utils.LongNanoID()
		if err != nil {
			err = fmt.Errorf("generating nanoID: %w", err)
			return
		}
	}

	err = work.validate()
	if err != nil {
		err = fmt.Errorf("validating work: %w", err)
		return
	}

	if work.Resume {
		err = p.db.UpdateWorkflowInstanceStatusByID(ctx, work.ID, model.WorkflowInstanceStatusScheduled, model.WorkflowInstanceStatusPaused)
		if err != nil {
			err = fmt.Errorf("updating workflow instance status from paused to scheduled: %w", err)
			return
		}
	} else {
		instance := model.WorkflowInstance{
			ID:          work.ID,
			WorkflowID:  work.WorkflowID,
			Status:      model.WorkflowInstanceStatusScheduled,
			StartNodeID: work.StartNodeID,
		}
		err = p.db.InsertWorkflowInstance(ctx, &instance)
		if err != nil {
			err = fmt.Errorf("inserting WorkflowInstance: %w", err)
			return
		}
	}

	marshaled, err := json.Marshal(work)
	if err != nil {
		err = fmt.Errorf("marshaling work into JSON: %w", err)
		return
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		// we use round-robin, so the key is not required.
		Key:   nil,
		Value: marshaled,
	})
	if err != nil {
		err = fmt.Errorf("pushing work into Kafka: %w", err)
		return
	}

	return
}
