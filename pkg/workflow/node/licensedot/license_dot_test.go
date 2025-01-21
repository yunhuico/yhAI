package licensedot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func TestCredentialTemplate(t *testing.T) {
	manager := adapter.GetAdapterManager()
	meta := manager.LookupAdapter("ultrafox/licensedot")
	meta.TestCredentialTemplate(adapter.AccessTokenCredentialType, map[string]any{
		"accessToken": "xxxxx",
	})
}

type testAuthorizer struct{}

func (t testAuthorizer) GetAccessToken(ctx context.Context) (string, error) {
	return "test-token", nil
}

func (t testAuthorizer) DecodeMeta(meta interface{}) error {
	data := []byte(` {
		"server":"https://license.com",
		"email":"test@test.com",
		"accessToken":"test-token"
	}`)

	return json.Unmarshal(data, meta)
}

func (t testAuthorizer) DecodeTokenMetaData(ctx context.Context, meta interface{}) error {
	return nil
}

func (t testAuthorizer) CredentialType() model.CredentialType {
	return model.CredentialTypeCustom
}

func TestCreateLicense(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(http.MethodPost, "https://license.com/licenses", func(request *http.Request) (*http.Response, error) {
		data := []byte(`{"id":33,"name":"test-01","users_count":100}`)
		resp := httpmock.NewBytesResponse(200, data)
		return resp, nil
	})

	ctx := context.Background()
	auth := &testAuthorizer{}
	client, err := newClient(ctx, auth)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	value := url.Values{
		"name": {"test1"},
		"code": {"test2"},
	}

	resp, err := client.createLicense(ctx, value)
	assert.NoError(t, err)
	assert.Equal(t, resp, []byte(`{"id":33,"name":"test-01","users_count":100}`))
}
