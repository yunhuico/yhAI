package apiclient

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
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
)

// sessionCookieKey is where session cookie lies on browsers
const sessionCookieKey = "fox_sess"

// Client api server client for request api.
// Maybe we can find a tool generate client code by swagger.
type Client struct {
	remoteURL url.URL
	session   string
}

func NewClient(opt ClientOpt) (client *Client, err error) {
	parsed, err := url.Parse(opt.RemoteURL)
	if err != nil {
		err = fmt.Errorf("parsing remoteURL: %w", err)
		return
	}
	if parsed.Host == "" {
		err = errors.New("host is missing in remoteURL")
		return
	}
	if parsed.Scheme == "" {
		err = errors.New("scheme is missing in remoteURL")
		return
	}

	if opt.Session == "" {
		err = errors.New("session is empty")
		return
	}

	client = &Client{
		remoteURL: url.URL{
			Scheme: parsed.Scheme,
			Host:   parsed.Host,
		},
		session: opt.Session,
	}
	return
}

type ClientOpt struct {
	RemoteURL string
	Session   string
}

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// request abstract away request forge and authorization.
// body can be io.Reader or anything that can be marshaled into JSON.
// respDest must be a pointer or nil.
func (c *Client) do(ctx context.Context, method string, path string, header map[string]string, query map[string]string, body any, respDest any) (err error) {
	reqURL := c.remoteURL
	reqURL.Path = path

	if len(query) > 0 {
		values := make(url.Values, len(query))
		for k, v := range query {
			values.Set(k, v)
		}
		reqURL.RawQuery = values.Encode()
	}
	var bodyReader io.Reader = http.NoBody
	if body != nil {
		if bodyAsIOReader, ok := body.(io.Reader); ok {
			bodyReader = bodyAsIOReader
		} else {
			var rawBody []byte
			rawBody, err = json.Marshal(body)
			if err != nil {
				err = fmt.Errorf("marsharling body into JSON: %w", err)
				return
			}
			bodyReader = bytes.NewReader(rawBody)
		}
	}

	req, err := http.NewRequest(method, reqURL.String(), bodyReader)
	if err != nil {
		err = fmt.Errorf("initializing http request: %w", err)
		return
	}
	req = req.WithContext(ctx)
	for key, value := range header {
		req.Header.Add(key, value)
	}
	req.AddCookie(&http.Cookie{
		Name:     sessionCookieKey,
		Value:    c.session,
		Expires:  time.Now().Add(time.Hour),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("making http request: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rawRespBody := make([]byte, 512)
		_, _ = io.ReadFull(resp.Body, rawRespBody)
		err = fmt.Errorf("remote responded with status code %d, body: %s", resp.StatusCode, rawRespBody)
		return
	}
	decoder := json.NewDecoder(resp.Body)

	var envelope envelope
	err = decoder.Decode(&envelope)
	if err != nil {
		err = fmt.Errorf("decoding response into envelope: %w", err)
		return
	}
	if envelope.Code != 0 {
		err = fmt.Errorf("remote response with code %d, message: %s", envelope.Code, envelope.Msg)
		return
	}
	if respDest != nil {
		err = json.Unmarshal(envelope.Data, respDest)
		if err != nil {
			err = fmt.Errorf("decoding JSON into respDest(type: %T): %w", respDest, err)
			return
		}
	}
	return
}

func (c *Client) ListCredentials(ctx context.Context, offset, limit int) (credentials response.ListCredentialResp, err error) {
	query := map[string]string{
		"offset": strconv.Itoa(offset),
		"limit":  strconv.Itoa(limit),
	}

	err = c.do(ctx, http.MethodGet, "/api/v1/credentials", nil, query, nil, &credentials)
	return
}

func (c *Client) ApplyWorkflow(ctx context.Context, content io.Reader) (workflowID string, err error) {
	err = c.do(ctx, http.MethodPut, "/api/v1/workflows/apply", nil, nil, content, &workflowID)
	return
}

func (c *Client) ExportWorkflow(ctx context.Context, workflowID string) (exportedWorkflow response.ExportWorkflowResp, err error) {
	err = c.do(ctx, http.MethodPost, fmt.Sprintf("api/v1/workflows/share/%s/export", workflowID), nil, nil, nil, &exportedWorkflow)
	return
}

func (c *Client) ImportWorkflow(ctx context.Context, content io.Reader, contentType string) (workflowID string, err error) {
	header := map[string]string{"Content-Type": contentType}
	err = c.do(ctx, http.MethodPost, "api/v1/workflows/share/import", header, nil, content, &workflowID)
	return
}
