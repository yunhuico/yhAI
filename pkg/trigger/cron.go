package trigger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const (
	numberOfRetry = 2
	dkronFail     = "fail"
	dkronSuccess  = "success"
)

type DkronOpt struct {
	// dkron internal host, e.g: http(s)://example.com:port
	Dkron string
	// webhook internal host, e.g: http(s)://example.com:port
	Webhook string
}

type DkronConfig struct {
	// internal Dkron address like http://localhost:8082
	DkronInternalHost string `comment:"internal Dkron address like http://localhost:8082"`
	// internal webhook address like http://localhost:8080
	WebhookInternalHost string `comment:"internal webhook address like http://localhost:8080"`
	// job tags to ensure jobs are scheduled to only one node
	//
	// Ref: https://dkron.io/docs/usage/target-nodes-spec#operation/showExecutionByID
	JobTags map[string]string `comment:"job tags to ensure jobs are scheduled to only one node"`
}

type DkronClient struct {
	dkronInternalHost   string
	webhookInternalHost string
	jobTags             map[string]string
}

type HTTPExecutor struct {
	Method     string `json:"method"`
	URL        string `json:"url"`
	Headers    string `json:"headers"`
	Body       string `json:"body"`
	ExpectCode string `json:"expectCode"`
}

type DkronJob struct {
	Name           string       `json:"name"`
	Schedule       string       `json:"schedule"`
	Timezone       string       `json:"timezone,omitempty"`
	Executor       string       `json:"executor"`
	ExecutorConfig HTTPExecutor `json:"executor_config"` // nolint

	// Optional
	Tags     map[string]string `json:"tags,omitempty"`
	Status   string            `json:"status,omitempty"`
	Disabled bool              `json:"disabled,omitempty"`
	Retries  int               `json:"retries,omitempty"`
}

// newDkronJob create a Dkron HTTP job.
//
// By default, all jobs are executed on every Dkron node in the cluster,
// which is usually not what we want.
// We can use nodeTags to enforce job execution to be scheduled to at most one node.
// Ref: https://dkron.io/docs/usage/target-nodes-spec#operation/showExecutionByID
func newDkronJob(triggerID, expr, webhookURL, timezone string, nodeTags map[string]string, httpBody string) DkronJob {
	return DkronJob{
		Name:     triggerID,
		Schedule: expr,
		Timezone: timezone,
		Executor: "http",
		ExecutorConfig: HTTPExecutor{
			Method:     http.MethodPost,
			URL:        webhookURL,
			Headers:    "[\"Content-Type: application/json\"]",
			Body:       httpBody,
			ExpectCode: strconv.Itoa(http.StatusOK),
		},
		Tags:     nodeTags,
		Retries:  numberOfRetry,
		Disabled: false,
	}
}

func (c *DkronClient) DeleteJob(ctx context.Context, triggerID string) error {
	respondedJob := &DkronJob{}
	err := c.call(ctx, http.MethodDelete, fmt.Sprintf("/jobs/%s", triggerID), nil, respondedJob)
	if errors.Is(err, errDkronJobNotFound) {
		return nil
	}

	return err
}

func (c *DkronClient) UpsertJob(ctx context.Context, triggerID, expr, timezone string, httpBody any) (*DkronJob, error) {
	webhookURL := fmt.Sprintf("%s/hooks/%s", c.webhookInternalHost, triggerID)

	var body string
	if httpBody != nil {
		marshaled, err := json.Marshal(httpBody)
		if err != nil {
			err = fmt.Errorf("marshaling HTTP body: %w", err)
			return nil, err
		}

		body = string(marshaled)
	}

	job := newDkronJob(triggerID, expr, webhookURL, timezone, c.jobTags, body)
	marshaledJob, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("marshalling dkron job: %w", err)
	}

	respondedJob := &DkronJob{}
	err = c.call(ctx, http.MethodPost, "/jobs", marshaledJob, respondedJob)
	if err != nil {
		return nil, err
	}

	if respondedJob.Status == dkronFail {
		return respondedJob, errors.New("creating dkron job failed")
	}
	return respondedJob, nil
}

func (c *DkronClient) ToggleJob(ctx context.Context, triggerID string) error {
	respondedJob := &DkronJob{}
	err := c.call(ctx, http.MethodPost, fmt.Sprintf("/jobs/%s/toggle", triggerID), nil, respondedJob)
	if err != nil {
		return err
	}
	if !respondedJob.Disabled {
		return errors.New("dkron job failed to be toggled")
	}
	return nil
}

func (c *DkronClient) RunJob(ctx context.Context, triggerID string) error {
	respondedJob := &DkronJob{}
	err := c.call(ctx, http.MethodPost, fmt.Sprintf("/jobs/%s", triggerID), nil, respondedJob)
	if err != nil {
		return err
	}

	if respondedJob.Disabled || respondedJob.Status != dkronSuccess {
		return errors.New("dkron job failed to run")
	}
	return nil
}

func (c *DkronClient) GetJobs(ctx context.Context) ([]DkronJob, error) {
	var jobs []DkronJob
	err := c.call(ctx, http.MethodGet, "/jobs", nil, &jobs)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

var errDkronJobNotFound = errors.New("job not found")

func (c *DkronClient) call(ctx context.Context, httpMethod, endpoint string, payload []byte, respData any) error {
	dkronURL := fmt.Sprintf("%s/v1%s", c.dkronInternalHost, endpoint)

	req, err := http.NewRequest(httpMethod, dkronURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("initializing http request for server %q: %w", c.dkronInternalHost, err)
	}

	req.Header.Set("Content-type", "application/json")
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errDkronJobNotFound
	}
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusAccepted {
		buf := make([]byte, 512)
		n, _ := io.ReadFull(resp.Body, buf)
		return fmt.Errorf("dkron response with status %s, body: %s", resp.Status, buf[:n])
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(respData)
	if err != nil {
		return fmt.Errorf("unmarshalling dkron job: %w", err)
	}

	return nil
}

func NewDkronClient(config DkronConfig) (client *DkronClient, err error) {
	parsedDkronURL, err := url.Parse(config.DkronInternalHost)
	if err != nil {
		err = fmt.Errorf("parsing dkron url: %w", err)
		return
	}
	if parsedDkronURL.Scheme == "" {
		err = fmt.Errorf("scheme of dkron %q is missing", config.DkronInternalHost)
		return
	}
	if parsedDkronURL.Host == "" {
		err = fmt.Errorf("host of dkron %q is missing", config.DkronInternalHost)
		return
	}

	parsedWebhookURL, err := url.Parse(config.WebhookInternalHost)
	if err != nil {
		err = fmt.Errorf("parsing webhook: %w", err)
		return
	}
	if parsedWebhookURL.Scheme == "" {
		err = fmt.Errorf("scheme of webhook %q is missing", config.WebhookInternalHost)
		return
	}
	if parsedWebhookURL.Host == "" {
		err = fmt.Errorf("host of webhook %q is missing", config.WebhookInternalHost)
		return
	}

	if len(config.JobTags) == 0 {
		err = errors.New("job tags are empty, which is usually a config error")
		return
	}

	client = &DkronClient{
		dkronInternalHost:   fmt.Sprintf("%s://%s", parsedDkronURL.Scheme, parsedDkronURL.Host),
		webhookInternalHost: fmt.Sprintf("%s://%s", parsedWebhookURL.Scheme, parsedWebhookURL.Host),
		jobTags:             config.JobTags,
	}

	return
}
