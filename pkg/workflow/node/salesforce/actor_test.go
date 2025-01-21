package salesforce

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://test.salesforce.com/services/oauth2/userinfo",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, []byte(userJson))
			return resp, nil
		},
	)

	userInfo := newUserInfo(t)
	contactAPI := userInfo.sobjectURL() + "Contact"
	accountAPI := userInfo.sobjectURL() + "Account"
	httpmock.RegisterResponder("GET", `=~^`+accountAPI+`/(\d+)\z`, func(request *http.Request) (*http.Response, error) {
		id, _ := httpmock.GetSubmatch(request, 1)
		if id != "12" {
			return httpmock.NewBytesResponse(http.StatusNotFound, []byte(`[{"message":"not found","errorCode":"404"}]`)), nil
		}
		return httpmock.NewBytesResponse(http.StatusOK, []byte("{\"accountId\":\"12\",\"name\":\"test-12\"}")), nil
	})

	httpmock.RegisterResponder("GET", `=~^`+contactAPI+`/(\d+)\z`, func(request *http.Request) (*http.Response, error) {
		id, _ := httpmock.GetSubmatch(request, 1)
		if id != "12" {
			return httpmock.NewBytesResponse(http.StatusNotFound, []byte(`[{"message":"not found","errorCode":"404"}]`)), nil
		}
		return httpmock.NewBytesResponse(http.StatusOK, []byte("{\"contactId\":\"13\",\"name\":\"test-13\"}")), nil
	})

	t.Run("query account", func(t *testing.T) {
		account := QueryAccountByID{
			AccountID: "12",
			BaseNode: BaseNode{
				client: NewClient("test-token", "https://test.salesforce.com"),
			},
		}
		res, err := account.Run(nil)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		v, ok := res.(map[string]any)
		assert.True(t, ok, true)
		assert.NotNil(t, v["name"])
		assert.Equal(t, v["name"], "test-12")

		errAccount := QueryAccountByID{
			AccountID: "13",
			BaseNode: BaseNode{
				client: NewClient("test-token", "https://test.salesforce.com"),
			},
		}

		res, err = errAccount.Run(nil)
		assert.NotNil(t, err)
		assert.Nil(t, res)
		assert.True(t, strings.Contains(err.Error(), "not found"))
	})

	t.Run("query contact", func(t *testing.T) {
		contact := QueryContactByID{
			ContactID: "12",
			BaseNode: BaseNode{
				client: NewClient("test-token", "https://test.salesforce.com"),
			},
		}
		res, err := contact.Run(nil)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		v, ok := res.(map[string]any)
		assert.True(t, ok, true)
		assert.NotNil(t, v["name"])
		assert.Equal(t, v["name"], "test-13")

		errContact := QueryContactByID{
			ContactID: "13",
			BaseNode: BaseNode{
				client: NewClient("test-token", "https://test.salesforce.com"),
			},
		}

		res, err = errContact.Run(nil)
		assert.NotNil(t, err)
		assert.Nil(t, res)
		assert.True(t, strings.Contains(err.Error(), "not found"))
	})
}
