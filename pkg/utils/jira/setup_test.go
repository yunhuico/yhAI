package jira

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

var Suite = &ClientSuite{}

type ClientSuite struct {
	once   sync.Once
	client *Client
}

func (s *ClientSuite) Run(t *testing.T, test func(t *testing.T, client *Client, ctx context.Context)) {
	t.Helper()

	if os.Getenv("TEST_JIRA_CLIENT") == "" {
		t.Skip()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("Jira", func(t *testing.T) {
		test(t, s.Client(), ctx)
	})
}

func (s *ClientSuite) Client() *Client {
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	s.once.Do(func() {
		s.client, err = NewClient(Config{
			AccountEmail: os.Getenv("JIRA_EMAIL"),
			APIToken:     os.Getenv("JIRA_TOKEN"),
			BaseURL:      os.Getenv("JIRA_BASE_URL"),
		})
		if err != nil {
			err = fmt.Errorf("creating client: %w", err)
			return
		}
	})

	return s.client
}
