// Package jira implements a subset of Jira REST API v3
// described at https://developer.atlassian.com/cloud/jira/platform/rest/v3/intro/
//
// The client uses Atlassian user API token generated at
// https://id.atlassian.com/manage-profile/security/api-tokens
package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Config struct {
	// user's Jira account email
	AccountEmail string
	// generated at https://id.atlassian.com/manage-profile/security/api-tokens
	APIToken string
	// e.g. https://your-domain.atlassian.net/ or https://issues.apache.org/jira/
	BaseURL string
}

type Client struct {
	client *http.Client

	// e.g. https://your-domain.atlassian.net/ or https://issues.apache.org/jira/
	baseURL string
	// encoded basic auth header like "Basic c29tZW9uZUBleGFtcGxlLmNvbTpwYXNzd29yZAo="
	basicAuth string
}

func NewClient(config Config) (client *Client, err error) {
	if config.AccountEmail == "" {
		err = errors.New("account email is missing")
		return
	}
	if config.APIToken == "" {
		err = errors.New("API token is empty")
		return
	}
	if config.BaseURL == "" {
		err = errors.New("base URL is empty")
		return
	}
	parsedBaseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		err = fmt.Errorf("parsing base URL: %w", err)
		return
	}
	if parsedBaseURL.Scheme != "https" {
		err = fmt.Errorf("scheme of base URL %q must be https, got %q", config.BaseURL, parsedBaseURL.Scheme)
		return
	}

	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", config.AccountEmail, config.APIToken)))

	client = &Client{
		client:    &http.Client{},
		basicAuth: basicAuth,
		baseURL:   strings.TrimRight(fmt.Sprintf("%s://%s%s", parsedBaseURL.Scheme, parsedBaseURL.Host, parsedBaseURL.Path), "/"),
	}

	return
}

type callOpt struct {
	// GET, POST, etc...
	Method string
	// e.g. /rest/api/3/search
	Path string
	// anything to be marshaled into JSON, leave nil to omit
	Body any
	// query to included in the request
	Query url.Values
	// where the result JSON to be unmarshalled to
	Dest any
}

func (c *Client) call(ctx context.Context, opt callOpt) (err error) {
	var bodyReader io.Reader = http.NoBody
	if opt.Body != nil {
		var marshaled []byte
		marshaled, err = json.Marshal(opt.Body)
		if err != nil {
			err = fmt.Errorf("marshaling body into JSON: %w", err)
			return
		}

		bodyReader = bytes.NewReader(marshaled)
	}

	var urlBuf strings.Builder

	urlBuf.WriteString(c.baseURL)
	urlBuf.WriteString(opt.Path)

	if len(opt.Query) > 0 {
		urlBuf.WriteString("?")
		urlBuf.WriteString(opt.Query.Encode())
	}

	req, err := http.NewRequest(opt.Method, urlBuf.String(), bodyReader)
	if err != nil {
		err = fmt.Errorf("init request: %w", err)
		return
	}
	req = req.WithContext(ctx)

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Authorization", c.basicAuth)
	req.Header.Set("User-Agent", "Jira Client by UltraFox")
	if opt.Body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		err = fmt.Errorf("making HTTP request: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := make([]byte, 512)
		n, _ := io.ReadFull(resp.Body, message)
		err = fmt.Errorf("remote responses with status %s, message: %s", resp.Status, message[:n])
		return
	}
	if opt.Dest != nil {
		err = json.NewDecoder(resp.Body).Decode(&opt.Dest)
		if err != nil {
			err = fmt.Errorf("unmarshaling response body: %w", err)
			return
		}
	}

	return
}

type RespPagination struct {
	// the index of the first item returned in the page.
	StartAt int `json:"startAt,omitempty"`
	// the maximum number of items that a page can return.
	// Each operation can have a different limit for the number of items returned, and these limits may change without notice.
	MaxResults int `json:"maxResults,omitempty"`
	// the total number of items contained in all pages.
	// This number may change as the client requests the subsequent pages,
	// therefore the client should always assume that the requested page can be empty.
	// Note that this property is not returned for all operations.
	Total int `json:"total,omitempty"`
}
