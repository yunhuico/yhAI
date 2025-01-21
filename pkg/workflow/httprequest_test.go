package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_httpRequest_Run(t *testing.T) {
	r := &HTTPRequest{
		Header: nil,
		Query:  nil,
		Body:   []byte(`"http"`),
	}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	require.Equal(t, "{\"header\":null,\"query\":null,\"body\":\"Imh0dHAi\"}", string(b))
}
