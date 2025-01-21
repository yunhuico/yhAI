package gitlab

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func TestGitlabCredentials(t *testing.T) {
	manager := adapter.GetAdapterManager()
	meta := manager.LookupAdapter("ultrafox/gitlab")
	meta.TestCredentialTemplate(adapter.AccessTokenCredentialType, map[string]any{
		"accessToken": "xxxxx",
	})
}

func Test_testCredential(t *testing.T) {
	ctx := context.Background()
	t.Run("test oauth2 credential", func(t *testing.T) {
		assert.NoError(t, testCredential(ctx, model.CredentialTypeOAuth, nil))
	})

	t.Run("accessToken empty", func(t *testing.T) {
		assert.Error(t, testCredential(ctx, model.CredentialTypeAccessToken, map[string]any{"server": "http://localhost:8021"}))
	})

	t.Run("server empty", func(t *testing.T) {
		assert.Error(t, testCredential(ctx, model.CredentialTypeAccessToken, map[string]any{"accessToken": "token"}))
	})

	t.Run("test pass", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://localhost:8021/api/v4/version",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewBytesResponse(200, []byte(`{}`))
				return resp, nil
			},
		)

		assert.NoError(t, testCredential(ctx, model.CredentialTypeAccessToken, map[string]any{
			"server":      "http://localhost:8021",
			"accessToken": "token",
		}))
	})

	t.Run("test failed", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://localhost:8021/api/v4/version",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(http.StatusUnauthorized, "Unauthorized")
				return resp, nil
			},
		)

		assert.Error(t, testCredential(ctx, model.CredentialTypeAccessToken, map[string]any{
			"server":      "http://localhost:8021",
			"accessToken": "token",
		}))
	})
}
