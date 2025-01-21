package salesforce

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapterMeta := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapterMeta.RegisterSpecsByDir(adapterDir)

	adapterMeta.RegisterCredentialTemplate(adapter.OAuth2CredentialType, `{
	"metaData": {
		{{ if eq .env "sandbox" }}
		"server": "https://test.salesforce.com"
		{{ else }}
		"server": "https://login.salesforce.com"
		{{ end }}
	},
	"oauth2Config": {
		"clientId": "{{ .clientId }}",
		"clientSecret": "{{ .clientSecret }}",
		"redirectUrl": "{{ .redirectUrl }}",
		"scopes": ["api", "refresh_token"],
		"endpoint": {
			{{ if eq .env "sandbox" }}
			"authUrl": "https://test.salesforce.com/services/oauth2/authorize",
			"tokenUrl": "https://test.salesforce.com/services/oauth2/token"
			{{ else }}
			"authUrl": "https://login.salesforce.com/services/oauth2/authorize",
			"tokenUrl": "https://login.salesforce.com/services/oauth2/token"
			{{ end }}
		}
	}
}`)

	workflow.RegistryNodeMeta(&AccountCreatedTrigger{})
	workflow.RegistryNodeMeta(&AccountUpdatedTrigger{})
	workflow.RegistryNodeMeta(&OpportunityCreatedTrigger{})
	workflow.RegistryNodeMeta(&OpportunityUpdatedTrigger{})
	workflow.RegistryNodeMeta(&ListContact{})
	workflow.RegistryNodeMeta(&ListAccount{})
	workflow.RegistryNodeMeta(&LicenseCreateTrigger{})
	workflow.RegistryNodeMeta(&QueryAccountByID{})
	workflow.RegistryNodeMeta(&QueryContactByID{})
}

//go:embed trigger.apex
var triggerCode string

//go:embed trigger_test.apex
var triggerTestCode string

//go:embed webhook.apex
var webhookCode string

//go:embed metaSoap.xml
var metaSoap string

var ErrDuplicateClass = errors.New("duplicate Apex class")

type Record map[string]any

func (r Record) GetID() string {
	v, ok := r["Id"].(string)
	if ok {
		return v
	}
	return ""
}

func (r Record) GetVersion() string {
	return r.GetID()
}

type queryResponse struct {
	Done      bool     `json:"done"`
	TotalSize int      `json:"totalSize"`
	Records   []Record `json:"records"`
}

type BaseNode struct {
	client *Client
}

func (b *BaseNode) Provision(ctx context.Context, dependencies workflow.ProvisionDeps) error {
	accessToken, err := dependencies.Authorizer.GetAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("getting salesforce access token: %w", err)
	}
	meta := &salesforceServerMeta{}
	err = dependencies.Authorizer.DecodeMeta(meta)
	if err != nil {
		return fmt.Errorf("decode auth meta data: %w", err)
	}
	b.client = NewClient(accessToken, meta.Server)
	return nil
}

// Client manages communication with the Salesforce API.
type Client struct {
	accessToken string
	apiServer   string
}

func NewClient(accessToken, apiServer string) *Client {
	return &Client{
		accessToken: accessToken,
		apiServer:   apiServer,
	}
}

func (c *Client) GetUserInfo() (*SFDCUserInfo, error) {
	userInfoUrl := c.apiServer + "/services/oauth2/userinfo"
	req, err := http.NewRequest("GET", userInfoUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("new http request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make http request: %w", err)
	}
	defer resp.Body.Close()

	var userInfo SFDCUserInfo
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}
	return &userInfo, nil
}

func (c *Client) triggerDelete(t trigger.WebhookContext) error {
	triggerData := t.GetTriggerData()
	apexTriggerClassId := getString(triggerData, APEXTRIGGERKEY)
	apexTestClassId := getString(triggerData, APEXTRIGGERTESTKEY)

	if apexTriggerClassId == "" {
		return nil
	}

	userInfo, err := c.GetUserInfo()
	if err != nil {
		return fmt.Errorf("get user info: %w", err)
	}

	restUrl := userInfo.restURL()
	url := restUrl + "tooling/sobjects/ApexTrigger/" + apexTriggerClassId
	err = c.deleteApexCode(url)
	if err != nil {
		return fmt.Errorf("delete triggger %w", err)
	}

	url = restUrl + "tooling/sobjects/ApexClass/" + apexTestClassId
	err = c.deleteApexCode(url)
	if err != nil {
		return fmt.Errorf("delete trigger test %w", err)
	}
	return nil
}

func (c *Client) triggerCreate(config *salesforceEventConfig) (map[string]any, error) {
	userInfo, err := c.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	apexTriggerCode, err := GenerateApexCode(config, "trigger", triggerCode)
	if err != nil {
		return nil, fmt.Errorf("generate apex trigger: %w", err)
	}
	apexTriggerTestCode, err := GenerateApexCode(config, "triggerTest", triggerTestCode)
	if err != nil {
		return nil, fmt.Errorf("generate apex test code: %w", err)
	}

	soap, err := GenerateApexCode(config, "soapMeta", metaSoap)
	if err != nil {
		return nil, fmt.Errorf("generate soap xml: %w", err)
	}

	webhookClassId, err := c.createApexClass(userInfo, "webhook", webhookCode)
	// webhook apex class can be reused
	if err != nil && err != ErrDuplicateClass {
		return nil, fmt.Errorf("create apex class on salesforce: %w", err)
	}

	err = c.createRemoteSite(userInfo, soap)
	if err != nil {
		return nil, fmt.Errorf("set remote site settings on salesforce: %w", err)
	}

	triggerClassId, err := c.createApexTrigger(userInfo, apexTriggerCode, config)
	if err != nil {
		return nil, fmt.Errorf("create apex trigger class on salesforce: %w", err)
	}
	triggerTestClassId, err := c.createApexClass(userInfo, "trigger", apexTriggerTestCode)
	if err != nil && err == ErrDuplicateClass {
		restUrl := userInfo.restURL()
		url := restUrl + "tooling/sobjects/ApexTrigger/" + triggerClassId
		_ = c.deleteApexCode(url)
		return nil, fmt.Errorf("create apex class on salesforce: %w", err)
	}
	return map[string]any{
		APEXCLASSKEY:       webhookClassId,
		APEXTRIGGERKEY:     triggerClassId,
		APEXTRIGGERTESTKEY: triggerTestClassId,
		APEXCLASSNAME:      config.TriggerName,
	}, nil
}

func (c *Client) createRemoteSite(info *SFDCUserInfo, body string) error {
	req, err := http.NewRequest("POST", info.metaURL(), bytes.NewBuffer([]byte(body)))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Add("Content-Type", "text/xml")
	req.Header.Add("SOAPAction", "RemoteSiteSetting")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("make http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 400 {
		return errors.New("bad request")
	}
	return nil
}

func (c *Client) createApexTrigger(info *SFDCUserInfo, body string, config *salesforceEventConfig) (string, error) {
	data := map[string]any{}
	data["ApiVersion"] = APIVERSION
	data["Name"] = config.TriggerName
	data["TableEnumOrId"] = config.SObject
	data["Body"] = body
	dbytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("encode  trigger data: %w", err)
	}
	url := info.restURL() + "tooling/sobjects/ApexTrigger"
	return c.post(url, dbytes)
}

func (c *Client) createApexClass(info *SFDCUserInfo, name, body string) (string, error) {
	data := map[string]interface{}{}
	data["ApiVersion"] = APIVERSION
	data["Body"] = body
	data["Name"] = name

	dbytes, err := toJSON(data)
	if err != nil {
		return "", fmt.Errorf("encode apex class: %w", err)
	}

	url := info.restURL() + "tooling/sobjects/ApexClass"
	return c.post(url, dbytes)
}

func (c *Client) get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("make new request error: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		decoder := json.NewDecoder(resp.Body)
		var errInfo ErrorRespDataArray
		err = decoder.Decode(&errInfo)
		if err == nil {
			for _, queryErr := range errInfo {
				err = fmt.Errorf("response error: %s: %s", queryErr.ErrorCode, queryErr.Message)
			}
		} else {
			err = fmt.Errorf("response error: %d %s", resp.StatusCode, resp.Status)
		}
		return nil, err
	}

	return ioutil.ReadAll(resp.Body)
}

func (c *Client) post(url string, data []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("new http request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Add("Sforce-Query-Options", "batchSize=2000")
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("make http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 400 {
		decoder := json.NewDecoder(resp.Body)
		var errInfo ErrorRespDataArray
		err = decoder.Decode(&errInfo)
		if err != nil {
			return "", fmt.Errorf("decode error message: %w", err)
		}
		if len(errInfo) > 0 {
			if errInfo[0].ErrorCode == ErrorDuplicate {
				return "", ErrDuplicateClass
			}
		}
		return "", fmt.Errorf("create apex class or trigger unknown error")
	}

	decoder := json.NewDecoder(resp.Body)
	var successData SuccessRespData
	err = decoder.Decode(&successData)
	if err != nil {
		return "", fmt.Errorf("decode SuccessRespData: %w", err)
	}
	return successData.ID, nil
}

func (c *Client) deleteApexCode(url string) error {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("new http request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("make http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("salesfroce server response %s", resp.Status)
	}
	return nil
}

func (c *Client) querySObject(req *http.Request) ([]byte, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Add("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		decoder := json.NewDecoder(resp.Body)
		var errInfo ErrorRespDataArray
		err = decoder.Decode(&errInfo)
		var errMsg error
		if err == nil {
			for _, queryErr := range errInfo {
				errMsg = fmt.Errorf("response error: %s: %s", queryErr.ErrorCode, queryErr.Message)
			}
		} else {
			errMsg = fmt.Errorf("response error: %d %s", resp.StatusCode, resp.Status)
		}
		return nil, errMsg
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read data error: %w", err)
	}
	return data, nil
}

func (c *Client) getSObjectCollection(sobject string, like *likeSql, limit, offset int) (*queryResponse, error) {
	userInfo, err := c.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	queryUrl := userInfo.queryURL()
	q, err := newQuery(queryOpt{
		Fields: []string{
			"FIELDS(ALL)",
		},
		SObject: sobject,
		Limit:   limit,
		Offset:  offset,
		Like:    like,
	})
	sql, err := q.Format()
	if err != nil {
		return nil, fmt.Errorf("soql format error: %w", err)
	}
	form := url.Values{}
	form.Add("q", sql)
	queryUrl += "?" + form.Encode()
	req, err := http.NewRequest(http.MethodGet, queryUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("make http request: %w", err)
	}
	data, err := c.querySObject(req)
	if err != nil {
		return nil, fmt.Errorf("query sobject: %w", err)
	}
	var resp queryResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, fmt.Errorf("decode response error: %w", err)
	}
	return &resp, nil
}
