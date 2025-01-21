package salesforce

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
)

var userJson = `{
    "sub":"https://test.salesforce.com/id/00DEm0000001sxdMAA/005Em000000HMlhIAG",
    "user_id":"005Em000000HMlhIAG",
    "organization_id":"00DEm0000001sxdMAA",
    "preferred_username":"wyang@jihulab.com.test",
    "nickname":"User16615022198879952028",
    "name":"Salesforce Test",
    "email":"wyang@jihulab.com",
    "email_verified":true,
    "family_name":"Salesforce Test",
    "zoneinfo":"Asia/Shanghai",
    "photos":{
        "picture":"https://jihulab--test.sandbox.file.force.com/profilephoto/005/F",
        "thumbnail":"https://jihulab--test.sandbox.file.force.com/profilephoto/005/T"
    },
    "profile":"https://jihulab--test.sandbox.my.salesforce.com/005Em000000HMlhIAG",
    "picture":"https://jihulab--test.sandbox.file.force.com/profilephoto/005/F",
    "address":{
        "country":"China"
    },
    "urls":{
        "enterprise":"https://jihulab--test.sandbox.my.salesforce.com/services/Soap/c/{version}/00DEm0000001sxd",
        "metadata":"https://jihulab--test.sandbox.my.salesforce.com/services/Soap/m/{version}/00DEm0000001sxd",
        "partner":"https://jihulab--test.sandbox.my.salesforce.com/services/Soap/u/{version}/00DEm0000001sxd",
        "rest":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/",
        "sobjects":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/sobjects/",
        "search":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/search/",
        "query":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/query/",
        "recent":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/recent/",
        "tooling_soap":"https://jihulab--test.sandbox.my.salesforce.com/services/Soap/T/{version}/00DEm0000001sxd",
        "tooling_rest":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/tooling/",
        "profile":"https://jihulab--test.sandbox.my.salesforce.com/005Em000000HMlhIAG",
        "feeds":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/chatter/feeds",
        "groups":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/chatter/groups",
        "users":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/chatter/users",
        "feed_items":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/chatter/feed-items",
        "feed_elements":"https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/chatter/feed-elements",
        "custom_domain":"https://jihulab--test.sandbox.my.salesforce.com"
    },
    "active":true,
    "user_type":"STANDARD",
    "language":"zh_CN",
    "locale":"zh_CN",
    "utcOffset":28800000,
    "updated_at":"2022-09-28T06:07:51Z",
    "is_app_installed":true
}`

var errJson = `"abc":"abc"`

func TestClient_GetUserInfo(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	apiUrl := "http://mock_server/api/services/oauth2/userinfo"
	accessToken := "salesforceTest"
	uri := "http://mock_server/api"
	t.Run("test correct user info", func(t *testing.T) {
		httpmock.RegisterResponder("GET", apiUrl,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewBytesResponse(200, []byte(userJson))
				return resp, nil
			},
		)
		client := NewClient(accessToken, uri)
		userInfo, err := client.GetUserInfo()
		assert.Nil(t, err)
		assert.Equal(t, userInfo.Name, "Salesforce Test")
	})

	t.Run("test error user info", func(t *testing.T) {
		httpmock.RegisterResponder("GET", apiUrl, func(request *http.Request) (*http.Response, error) {
			return httpmock.NewBytesResponse(200, []byte(errJson)), nil
		})
		client := NewClient(accessToken, uri)
		userInfo, err := client.GetUserInfo()
		assert.NotNil(t, err)
		assert.Nil(t, userInfo)
	})
}

func TestQuerResponseAndRecord(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	queryUrl := "https://test_server.com/service/data"
	accessToken := "salesforceTest"
	uri := "https://test_server.com/service"
	t.Run("Test query response", func(t *testing.T) {
		httpmock.RegisterResponder("GET", queryUrl, func(request *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, []byte(`{"done": true, "totalSize": 1, "records":[{"Id":"123","Name":"test-01"}]}`))
			return resp, nil
		})

		client := NewClient(accessToken, uri)
		req, err := http.NewRequest("GET", queryUrl, nil)
		if err != nil {
			panic(err)
		}
		data, err := client.querySObject(req)
		assert.NoError(t, err)
		assert.NotNil(t, data)

		var resp queryResponse
		err = json.Unmarshal(data, &resp)
		assert.NoError(t, err)
		assert.Equal(t, resp.TotalSize, 1)
		assert.Greater(t, len(resp.Records), 0)
		assert.Equal(t, resp.Records[0].GetID(), "123")
		assert.Equal(t, resp.Records[0].GetVersion(), "123")
	})

	noExistUrl := "https://test_server.com/service/data/v1"
	t.Run("Test query response id not exist", func(t *testing.T) {
		httpmock.RegisterResponder("GET", noExistUrl, func(request *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, []byte(`{"done": true, "totalSize": 1, "records":[{"IID":"123","Name":"test-01"}]}`))
			return resp, nil
		})

		client := NewClient(accessToken, uri)
		req, err := http.NewRequest("GET", noExistUrl, nil)
		if err != nil {
			panic(err)
		}
		data, err := client.querySObject(req)
		assert.NoError(t, err)
		assert.NotNil(t, data)

		var resp queryResponse
		err = json.Unmarshal(data, &resp)
		assert.NoError(t, err)
		assert.Equal(t, resp.TotalSize, 1)
		assert.Greater(t, len(resp.Records), 0)
		assert.Equal(t, resp.Records[0].GetID(), "")
		assert.Equal(t, resp.Records[0].GetVersion(), "")
	})

}

func TestCredentialTemplate(t *testing.T) {
	manager := adapter.GetAdapterManager()
	meta := manager.LookupAdapter("ultrafox/salesforce")
	meta.TestCredentialTemplate(adapter.OAuth2CredentialType, map[string]any{
		"accessToken": "xxxxx",
	})
}
