package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HTTPRequest struct {
	Header map[string]string `json:"header"`
	Query  map[string]string `json:"query"`
	Body   []byte            `json:"body"`
}

func (h *HTTPRequest) Marshal() ([]byte, error) {
	return json.Marshal(h)
}

func BuildHTTPRequest(req *http.Request) (*HTTPRequest, error) {
	header := make(map[string]string, len(req.Header))
	for k := range req.Header {
		header[k] = req.Header.Get(k)
	}

	query := make(map[string]string, len(req.URL.Query()))
	for k := range req.URL.Query() {
		query[k] = req.URL.Query().Get(k)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	return &HTTPRequest{
		Header: header,
		Query:  query,
		Body:   body,
	}, nil
}
