package http

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapterMeta := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapterMeta.RegisterSpecsByDir(adapterDir)

	workflow.RegistryNodeMeta(&makeRequest{})
}

type kv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var allowMethods = set.FromSlice([]string{
	"GET",
	"POST",
	"DELETE",
	"PATCH",
	"PUT",
	"OPTIONS",
	"HEAD",
})

var allowBodyTypes = set.FromSlice([]string{
	"empty",
	"raw",
	"formData",
	"formUrlencoded",
})

var allowContentTypes = set.FromSlice([]string{
	"empty",
	"text",
	"json",
})

type makeRequest struct {
	Method      string `json:"method"`
	URL         string `json:"url"`
	Queries     []kv   `json:"queries"`
	Headers     []kv   `json:"headers"`
	BodyType    string `json:"bodyType"`
	ContentType string `json:"contentType"`
	// RawContent only available when BodyType == "raw"
	RawContent  string `json:"rawContent"`
	FormContent []kv   `json:"formContent"`
}

func (m makeRequest) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/http#makeRequest")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(makeRequest)
		},
		InputForm: spec.InputSchema,
	}
}

// TODO(sword): abstract the validation when creating or updating node.
func (m makeRequest) validate(updating bool) error {
	if !allowMethods.Has(m.Method) {
		return errors.New("method is invalid")
	}
	if !allowBodyTypes.Has(m.BodyType) {
		return errors.New("bodyType is invalid")
	}
	if m.BodyType == "raw" {
		if !allowContentTypes.Has(m.ContentType) {
			return errors.New("contentType is invalid")
		}
	}
	if m.URL == "" {
		return errors.New("url is invalid")
	}
	return nil
}

var (
	maxRequestTimeout         = 40 * time.Second
	maxResponseBodySize int64 = 2 << 20 // 2MB
)

type responseOutput struct {
	StatusCode int               `json:"statusCode"`
	Status     string            `json:"status"`
	Header     map[string]string `json:"header"`
	Body       any               `json:"body"`
}

func (m makeRequest) Run(c *workflow.NodeContext) (output any, err error) {
	ctx, cancel := context.WithTimeout(c.Context(), maxRequestTimeout)
	defer cancel()

	return m.run(ctx)
}

func (m makeRequest) run(ctx context.Context) (output responseOutput, err error) {
	if err = m.validate(false); err != nil {
		err = fmt.Errorf("validating node: %w", err)
		return
	}

	resp, err := m.request(ctx)
	if err != nil {
		err = fmt.Errorf("calling http request: %w", err)
		return
	}
	defer resp.Body.Close()

	output, err = m.readResponse(resp)
	if err != nil {
		err = fmt.Errorf("reading response: %w", err)
		return
	}
	return
}

func (m makeRequest) request(ctx context.Context) (response *http.Response, err error) {
	var (
		requestBody io.Reader
		contentType string
	)

	switch m.BodyType {
	case "empty":
		requestBody = nil
	case "raw":
		requestBody = strings.NewReader(m.RawContent)
		if m.ContentType == "json" {
			contentType = "application/json"
		} else if m.ContentType == "text" {
			contentType = "text/plain"
		} else if m.ContentType == "empty" {
			requestBody = nil
		}
	case "formUrlencoded":
		contentType = "application/x-www-form-urlencoded"
		formData := url.Values{}
		for _, formItem := range m.FormContent {
			if strings.TrimSpace(formItem.Key) == "" {
				continue
			}
			formData.Set(formItem.Key, formItem.Value)
		}
		requestBody = strings.NewReader(formData.Encode())
	case "formData":
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for _, formItem := range m.FormContent {
			if formItem.Key == "" {
				continue
			}
			err = writer.WriteField(formItem.Key, formItem.Value)
			if err != nil {
				err = fmt.Errorf("writing form field %s: %w", formItem.Key, err)
				return
			}
		}
		err = writer.Close()
		if err != nil {
			err = fmt.Errorf("writing form data: %w", err)
			return
		}
		requestBody = body
		contentType = writer.FormDataContentType()
	}

	var req *http.Request
	req, err = http.NewRequest(m.Method, m.URL, requestBody)
	if err != nil {
		err = fmt.Errorf("new request: %w", err)
		return
	}
	q := req.URL.Query()
	for _, query := range m.Queries {
		if strings.TrimSpace(query.Key) == "" {
			continue
		}
		q.Add(query.Key, query.Value)
	}
	req.URL.RawQuery = q.Encode()

	for _, header := range m.Headers {
		if strings.TrimSpace(header.Key) == "" {
			continue
		}
		req.Header.Set(header.Key, header.Value)
	}
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}
	req = req.WithContext(ctx)
	return http.DefaultClient.Do(req)
}

func (m makeRequest) readResponse(resp *http.Response) (output responseOutput, err error) {
	output.Status = resp.Status
	output.StatusCode = resp.StatusCode
	output.Header = buildHeader(resp.Header)

	limitedReader := io.LimitReader(resp.Body, maxResponseBodySize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		err = fmt.Errorf("reading response body: %w", err)
		return
	}
	var body any
	err = json.Unmarshal(responseBody, &body)
	if err != nil {
		body = string(responseBody)
		err = nil
	}
	output.Body = body
	return
}

func buildHeader(header http.Header) (output map[string]string) {
	if len(header) > 0 {
		output = make(map[string]string, len(header))
		for k := range header {
			output[k] = header.Get(k)
		}
	}
	return
}
