package licensedot

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapterMeta := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapterMeta.RegisterSpecsByDir(adapterDir)

	adapterMeta.RegisterCredentialTemplate(adapter.AccessTokenCredentialType, `{
	"metaData": {
		"server": "{{ .server }}",
		"email": "{{ .email }}"
	},
	"accessToken": "{{ .accessToken }}"
}`)

	workflow.RegistryNodeMeta(&CreateLicense{})
}

type licenseClient struct {
	accessToken string
	email       string
	server      string
}

type licenseAuthMeta struct {
	Server string `json:"server"`
	Email  string `json:"email"`
}

type BaseNode struct {
	client *licenseClient
}

func (b *BaseNode) Provision(ctx context.Context, dependencies workflow.ProvisionDeps) error {
	client, err := newClient(ctx, dependencies.Authorizer)
	if err != nil {
		return err
	}
	b.client = client
	return nil
}

func newClient(ctx context.Context, authorizer auth.Authorizer) (*licenseClient, error) {
	accessToken, err := authorizer.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	meta := licenseAuthMeta{}
	err = authorizer.DecodeMeta(&meta)
	if err != nil {
		return nil, fmt.Errorf("decode auth meta data: %w", err)
	}

	return &licenseClient{
		accessToken: accessToken,
		server:      meta.Server,
		email:       meta.Email,
	}, nil
}

func (c *licenseClient) createLicense(ctx context.Context, license url.Values) ([]byte, error) {
	data := license.Encode()
	req, err := http.NewRequest(http.MethodPost, c.server+"/licenses", strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("new http request error: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("X-User-Email", c.email)
	req.Header.Add("X-User-Token", c.accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make http request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf := make([]byte, 512)
		n, _ := io.ReadFull(resp.Body, buf)
		return nil, fmt.Errorf("license dot response with status %s, body: %s", resp.Status, buf[:n])
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response data error: %w", err)
	}
	return body, nil
}
