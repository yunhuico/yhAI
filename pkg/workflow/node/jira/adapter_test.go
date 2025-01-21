package jira

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func Test_testCredential(t *testing.T) {
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("invalid inputFields", func(t *testing.T) {
		err := testCredential(ctx, model.CredentialTypeAccessToken, map[string]any{})
		assert.Error(t, err)
	})

	t.Run("test pass", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://localhost:8021/rest/api/3/serverInfo",
			func(req *http.Request) (*http.Response, error) {
				resp, _ := httpmock.NewJsonResponse(200, map[string]string{})
				return resp, nil
			},
		)

		err := testCredential(ctx, model.CredentialTypeAccessToken, map[string]any{
			"accountEmail": "ultrafox@ultrafox.com",
			"apiToken":     "token",
			"baseUrl":      "https://localhost:8021",
		})
		assert.NoError(t, err)
	})

	t.Run("test fail", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "https://localhost:8021/rest/api/3/serverInfo",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(http.StatusUnauthorized, "Unauthorized")
				return resp, nil
			},
		)

		err := testCredential(ctx, model.CredentialTypeAccessToken, map[string]any{
			"accountEmail": "ultrafox@ultrafox.com",
			"apiToken":     "token",
			"baseUrl":      "https://localhost:8021",
		})
		assert.Error(t, err)
	})
}
