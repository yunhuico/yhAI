package work

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/segmentio/kafka-go"
)

// Suite takes care of creation of Kafka instance.
var Suite KafkaSuite

func TestMain(m *testing.M) { // nolint: staticcheck
	defer Suite.Close()

	// m.Run will return an exit code that may be passed to os.Exit.
	// If TestMain returns, the test wrapper will pass the result of m.Run to os.Exit itself.
	// ref: https://pkg.go.dev/testing#hdr-Main
	m.Run()
}

// KafkaSuite takes care of creation and disposal of Kafka instance.
type KafkaSuite struct {
	onceKafka      sync.Once
	kafkaClient    *kafka.Client
	kafkaAddresses []string
	purgeDocker    func()
}

func (s *KafkaSuite) Close() {
	if s.purgeDocker != nil {
		s.purgeDocker()
	}
}

// Run brings a working Kafka instance to test, the caller MUST NOT use the same topic between tests.
//
// To enable Kafka test, set ENV TEST_KAFKA=1
// To bring your own Kafka, set ENV KAFKA_ADDR=host1:port1[,host2:port2,...]
func (s *KafkaSuite) Run(t *testing.T, test func(t *testing.T, ctx context.Context, addresses []string, client *kafka.Client)) {
	t.Helper()

	if os.Getenv("TEST_KAFKA") != "1" {
		t.SkipNow()
	}

	addresses, client := s.Kafka()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	test(t, ctx, addresses, client)
}

func (s *KafkaSuite) Kafka() (addresses []string, client *kafka.Client) {
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	s.onceKafka.Do(func() {
		rawAddr := os.Getenv("KAFKA_ADDR")

		if rawAddr != "" {
			addresses = strings.Split(rawAddr, ",")
			err = pingKafka(ctx, addresses, SASLCredential{})
			if err != nil {
				err = fmt.Errorf("pinging Kafka: %w", err)
				return
			}
			client = &kafka.Client{
				Addr: kafka.TCP(addresses...),
			}

			s.kafkaAddresses = addresses
			s.kafkaClient = client

			return
		}

		var pool *dockertest.Pool
		pool, err = dockertest.NewPool("")
		if err != nil {
			err = fmt.Errorf("could not connect to docker: %w", err)
			return
		}
		// pulls an image, creates a container based on it and runs it
		var kafkaResource *dockertest.Resource
		kafkaResource, err = pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "martinnowak/kafka",
			Tag:        "2",
			Hostname:   "kafka",
			PortBindings: map[docker.Port][]docker.PortBinding{
				"19092/tcp": {{HostIP: "localhost", HostPort: "19092/tcp"}},
			},
			ExposedPorts: []string{"19092/tcp"},
			Entrypoint:   []string{"/bin/bash", "-c"},
			// ref: https://www.confluent.io/blog/kafka-listeners-explained/#:~:text=Let%E2%80%99s%20check%20out%20some%20config.
			Cmd: []string{`sed -i "s|^broker.id=.*$|broker.id=$BROKER_ID|" /opt/kafka/config/server.properties && \
    echo 'listener.security.protocol.map=INSIDE:PLAINTEXT,OUTSIDE:PLAINTEXT' >> /opt/kafka/config/server.properties  && \
    echo 'inter.broker.listener.name=INSIDE' >> /opt/kafka/config/server.properties  && \
    echo 'advertised.listeners=INSIDE://kafka:9092,OUTSIDE://localhost:19092' >> /opt/kafka/config/server.properties  && \
    echo 'listeners=INSIDE://0.0.0.0:9092,OUTSIDE://0.0.0.0:19092' >> /opt/kafka/config/server.properties  && \
    /opt/kafka/bin/zookeeper-server-start.sh -daemon /opt/kafka/config/zookeeper.properties && \
    exec /opt/kafka/bin/kafka-server-start.sh /opt/kafka/config/server.properties`},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		})
		if err != nil {
			err = fmt.Errorf("could not start kafkaResource: %w", err)
			return
		}
		err = kafkaResource.Expire(2 * 60)
		if err != nil {
			err = fmt.Errorf("[resource leaking] failed to set container expire: %w", err)
			return
		}
		s.purgeDocker = func() {
			err := pool.Purge(kafkaResource)
			if err != nil {
				err = fmt.Errorf("purging Kafka: %w", err)
				log.Printf("resource leaked: %s", err)
			}
		}

		addresses = []string{"localhost:19092"}
		client = &kafka.Client{
			Addr: kafka.TCP(addresses...),
		}
		s.kafkaAddresses = addresses
		s.kafkaClient = client

		// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
		err = pool.Retry(func() (err error) {
			err = pingKafka(ctx, addresses, SASLCredential{})
			return
		})
		if err != nil {
			err = fmt.Errorf("could not connect to Kafka container: %w", err)
			return
		}
	})
	if err != nil {
		return
	}

	fmt.Println("Kafka is up, time taken: ", time.Since(startedAt).String())
	return s.kafkaAddresses, s.kafkaClient
}
