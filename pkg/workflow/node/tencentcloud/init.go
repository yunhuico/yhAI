package tencentcloud

import (
	"context"
	"embed"
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapter := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapter.RegisterSpecsByDir(adapterDir)
	adapter.RegisterCredentialTestingFunc(testCredential)

	workflow.RegistryNodeMeta(&CvmDescribeInstances{})
	workflow.RegistryNodeMeta(&CvmStopInstances{})
	workflow.RegistryNodeMeta(&CvmTerminateInstances{})
}

func testCredential(ctx context.Context, _ model.CredentialType, fields model.InputFields) (err error) {
	secretID, ok := fields.GetString2("secretId", false)
	if !ok {
		err = fmt.Errorf("secretId is required")
		return
	}
	secretKey, ok := fields.GetString2("secretKey", false)
	if !ok {
		err = fmt.Errorf("secretKey is required")
		return
	}

	credential := common.NewCredential(secretID, secretKey)
	client, err := cvm.NewClient(credential, "ap-beijing", profile.NewClientProfile())
	if err != nil {
		err = fmt.Errorf("new cvm client: %w", err)
		return
	}

	req := cvm.NewDescribeZonesRequest()
	_, err = client.DescribeZones(req)
	if err != nil {
		err = fmt.Errorf("calling tencentcloud api: %w", err)
		return
	}

	return
}
