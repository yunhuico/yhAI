package salesforce

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
)

var metaData = `{"server":"https://test.salesforce.com"}`

var db = map[string]map[string]any{}

var _ trigger.WebhookContext = testContext{}

type testContext struct {
	log.Logger
	Id string
}

func (t testContext) GetPassportVendorLookup() map[model.PassportVendorName]model.PassportVendor {
	return nil
}

func (t testContext) SetTriggerQueryID(queryID string) {
}

type testAuthorizer struct{}

func (t testAuthorizer) DecodeTokenMetaData(ctx context.Context, meta interface{}) error {
	return nil
}

func (t testAuthorizer) GetAccessToken(ctx context.Context) (string, error) {
	return "test token", nil
}

func (t testAuthorizer) DecodeMeta(meta interface{}) error {
	return json.Unmarshal([]byte(metaData), meta)
}

func (t testAuthorizer) CredentialType() model.CredentialType {
	return model.CredentialTypeOAuth2
}

func (t testContext) Context() context.Context {
	return context.Background()
}

func (t testContext) GetConfigObject() any {
	return &salesforceEventConfig{}
}

func (t testContext) GetAuthorizer() auth.Authorizer {
	return &testAuthorizer{}
}

func (t testContext) GetTriggerData() map[string]any {
	if v, ok := db[t.Id]; ok {
		return v
	}
	return nil
}

func (t testContext) GetWebhookURL() string {
	randomId, err := utils.NanoID()
	if err != nil {
		return ""
	}
	return "https://test.salseforce.com/hooks/" + randomId
}

func newWebContext(t *testing.T) trigger.WebhookContext {
	log.Init("go-test.apiserver", log.DebugLevel)
	return &testContext{
		Logger: log.Clone(log.Namespace("salesforce/uint-test")),
	}
}

func TestGenerateApexCode(t *testing.T) {
	t.Run("test create trigger apex code", func(t *testing.T) {
		config := &salesforceEventConfig{
			SObject:     "account",
			Events:      []string{"after update"},
			SessionId:   "test-1",
			CallbackUrl: "https://test.ultrafox.com/callback",
		}

		code, err := GenerateApexCode(config, "test-1", webhookCode)
		assert.Nil(t, err)
		assert.True(t, true, strings.Contains(code, "after update"))
		assert.True(t, true, strings.Contains(code, "account"))
		assert.True(t, true, strings.Contains(code, config.CallbackUrl))
		assert.False(t, false, strings.Contains(code, config.SessionId))
	})

	t.Run("test create remote site settings", func(t *testing.T) {
		config := &salesforceEventConfig{
			SObject:     "account",
			Events:      []string{"after update"},
			SessionId:   "test-remote site settings",
			CallbackUrl: "https://test.ultrafox.com/callback",
		}
		code, err := GenerateApexCode(config, "test-remote site setting", metaSoap)
		assert.Nil(t, err)
		assert.True(t, true, strings.Contains(code, config.SessionId))
		assert.True(t, true, strings.Contains(code, config.CallbackUrl))
	})
}

func newUserInfo(t *testing.T) *SFDCUserInfo {
	return &SFDCUserInfo{
		URLs: SFDCURL{
			Rest:     "https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/",
			Metadata: "https://jihulab--test.sandbox.my.salesforce.com/services/Soap/m/{version}/00DEm0000001sxd",
			SObjects: "https://jihulab--test.sandbox.my.salesforce.com/services/data/v{version}/sobjects/",
		},
	}
}

func TestTrigger(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	webContext := newWebContext(t)
	userInfo := newUserInfo(t)
	apexApi := userInfo.restURL() + "tooling/sobjects/ApexClass"
	triggerApi := userInfo.restURL() + "tooling/sobjects/ApexTrigger"

	triggers := map[string]bool{}

	triggerCreate := func() []byte {
		id := rand.Intn(100)
		strconv.Itoa(id)
		result := map[string]any{
			"id":      strconv.Itoa(id),
			"success": true,
		}
		triggers[strconv.Itoa(id)] = true
		data, _ := json.Marshal(result)
		return data
	}

	triggerCheck := func(id string) bool {
		if _, ok := triggers[id]; ok {
			return true
		}
		return false
	}

	triggerDelete := func(id string) bool {
		if _, ok := triggers[id]; ok {
			delete(triggers, id)
			return true
		}
		return true
	}

	httpmock.RegisterResponder("GET", "https://test.salesforce.com/services/oauth2/userinfo",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, []byte(userJson))
			return resp, nil
		},
	)
	httpmock.RegisterResponder("POST", apexApi, func(request *http.Request) (*http.Response, error) {
		data := triggerCreate()
		resp := httpmock.NewBytesResponse(200, data)
		return resp, nil
	})
	httpmock.RegisterResponder("POST", triggerApi, func(request *http.Request) (*http.Response, error) {
		data := triggerCreate()
		resp := httpmock.NewBytesResponse(200, data)
		return resp, nil
	})
	httpmock.RegisterResponder("POST", userInfo.metaURL(), func(request *http.Request) (*http.Response, error) {
		data := triggerCreate()
		resp := httpmock.NewBytesResponse(200, data)
		return resp, nil
	})

	httpmock.RegisterResponder("GET", `=~^`+apexApi+`/(\d+)\z`, func(request *http.Request) (*http.Response, error) {
		id, _ := httpmock.GetSubmatch(request, 1)
		exist := triggerCheck(id)
		if exist {
			return httpmock.NewJsonResponse(http.StatusFound, "")
		}
		return httpmock.NewJsonResponse(http.StatusNotFound, "")
	})

	httpmock.RegisterResponder("DELETE", `=~^`+apexApi+`/(\d+)\z`, func(request *http.Request) (*http.Response, error) {
		id, _ := httpmock.GetSubmatch(request, 1)
		deleted := triggerDelete(id)
		if deleted {
			return httpmock.NewJsonResponse(http.StatusNoContent, "")
		}
		return httpmock.NewJsonResponse(http.StatusNotFound, "")
	})

	httpmock.RegisterResponder("GET", `=~^`+triggerApi+`/(\d+)\z`, func(request *http.Request) (*http.Response, error) {
		id, _ := httpmock.GetSubmatch(request, 1)
		exist := triggerCheck(id)
		if exist {
			return httpmock.NewJsonResponse(http.StatusNoContent, "")
		}
		return httpmock.NewJsonResponse(http.StatusNotFound, "")
	})

	httpmock.RegisterResponder("DELETE", `=~^`+triggerApi+`/(\d+)\z`, func(request *http.Request) (*http.Response, error) {
		id, _ := httpmock.GetSubmatch(request, 1)
		deleted := triggerDelete(id)
		if deleted {
			return httpmock.NewJsonResponse(http.StatusNoContent, "")
		}
		return httpmock.NewJsonResponse(http.StatusNotFound, "")
	})

	t.Run("test create account created trigger", func(t *testing.T) {
		accountCreatedTrigger := &AccountCreatedTrigger{}
		res, err := accountCreatedTrigger.Create(webContext)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res[APEXTRIGGERTESTKEY])
		assert.NotNil(t, res[APEXCLASSKEY])
		assert.NotNil(t, res[APEXTRIGGERKEY])
		assert.NotNil(t, res[APEXCLASSNAME])

		triggerId, _ := utils.NanoID()
		db[triggerId] = res
		tmpCxt := &testContext{Id: triggerId}

		tmpCxt = &testContext{Id: triggerId}
		err = accountCreatedTrigger.Delete(tmpCxt)
		assert.Nil(t, err)
	})

	t.Run("test create account updated trigger", func(t *testing.T) {
		accountUpdatedTrigger := &AccountUpdatedTrigger{}
		res, err := accountUpdatedTrigger.Create(webContext)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res[APEXTRIGGERTESTKEY])
		assert.NotNil(t, res[APEXCLASSKEY])
		assert.NotNil(t, res[APEXTRIGGERKEY])
		assert.NotNil(t, res[APEXCLASSNAME])

		triggerId, _ := utils.NanoID()
		db[triggerId] = res
		tmpCxt := &testContext{Id: triggerId}

		tmpCxt = &testContext{Id: triggerId}
		err = accountUpdatedTrigger.Delete(tmpCxt)
		assert.Nil(t, err)
	})

	t.Run("test create opportunity created trigger", func(t *testing.T) {
		oppCreated := OpportunityCreatedTrigger{}
		res, err := oppCreated.Create(webContext)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res[APEXTRIGGERTESTKEY])
		assert.NotNil(t, res[APEXCLASSKEY])
		assert.NotNil(t, res[APEXTRIGGERKEY])
		assert.NotNil(t, res[APEXCLASSNAME])

		triggerId, _ := utils.NanoID()
		db[triggerId] = res
		tmpCxt := &testContext{Id: triggerId}

		err = oppCreated.Delete(tmpCxt)
		assert.Nil(t, err)
	})
	t.Run("test create opportunity updated trigger", func(t *testing.T) {
		oppUpdated := OpportunityUpdatedTrigger{}
		res, err := oppUpdated.Create(webContext)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res[APEXTRIGGERTESTKEY])
		assert.NotNil(t, res[APEXCLASSKEY])
		assert.NotNil(t, res[APEXTRIGGERKEY])
		assert.NotNil(t, res[APEXCLASSNAME])

		triggerId, _ := utils.NanoID()
		db[triggerId] = res
		tmpCxt := &testContext{Id: triggerId}

		err = oppUpdated.Delete(tmpCxt)
		assert.Nil(t, err)
	})

	t.Run("test create license application trigger", func(t *testing.T) {
		license := LicenseCreateTrigger{}
		res, err := license.Create(webContext)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res[APEXTRIGGERTESTKEY])
		assert.NotNil(t, res[APEXCLASSKEY])
		assert.NotNil(t, res[APEXTRIGGERKEY])
		assert.NotNil(t, res[APEXCLASSNAME])

		triggerId, _ := utils.NanoID()
		db[triggerId] = res
		tmpCxt := &testContext{Id: triggerId}
		err = license.Delete(tmpCxt)
		assert.Nil(t, err)
	})
}
