package tencentcloud

import (
	"context"
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/collection"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type CvmDescribeInstances struct {
	Region  string        `json:"region"`
	Filters []*cvm.Filter `json:"filters"`
}

func (self *CvmDescribeInstances) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/tencentcloud#cvmDescribeInstances")

	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return &CvmDescribeInstances{}
		},
		InputForm: spec.InputSchema,
	}
}

var (
	limit int64 = 100
)

type CvmDescribeInstancesOutput struct {
	Instances   []*cvm.Instance `json:"instances"`
	InstanceIds []*string       `json:"instanceIds"`
	Total       int             `json:"total"`
}

func (self *CvmDescribeInstances) Run(c *workflow.NodeContext) (any, error) {
	client, err := newCvmClient(c.Context(), self.Region, c.GetAuthorizer())
	if err != nil {
		return nil, fmt.Errorf("new cvm client: %w", err)
	}

	output := CvmDescribeInstancesOutput{}

	request := cvm.NewDescribeInstancesRequest()
	request.Filters = self.Filters
	request.Limit = &limit

	var offset int64 = 0
	for {
		request.Offset = &offset
		response, err := client.DescribeInstances(request)
		if _, ok := err.(*errors.TencentCloudSDKError); ok {
			return nil, fmt.Errorf("An API error has returned: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("describe cvm instances: %w", err)
		}

		offset += int64(len(response.Response.InstanceSet))
		output.Instances = append(output.Instances, response.Response.InstanceSet...)
		for _, instance := range response.Response.InstanceSet {
			output.InstanceIds = append(output.InstanceIds, instance.InstanceId)
		}
		output.Total += len(response.Response.InstanceSet)
		if len(response.Response.InstanceSet) < int(limit) {
			break
		}
	}

	return output, nil
}

type CvmTerminateInstances struct {
	Region      string    `json:"region"`
	InstanceIds []*string `json:"instanceIds"`
}

func (self *CvmTerminateInstances) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/tencentcloud#cvmTerminateInstances")

	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return &CvmTerminateInstances{}
		},
		InputForm: spec.InputSchema,
	}
}

func (self *CvmTerminateInstances) Run(c *workflow.NodeContext) (any, error) {
	client, err := newCvmClient(c.Context(), self.Region, c.GetAuthorizer())
	if err != nil {
		return nil, fmt.Errorf("new cvm client: %w", err)
	}
	request := cvm.NewTerminateInstancesRequest()

	for _, groupIds := range collection.Chunk(self.InstanceIds, 100) {
		request.InstanceIds = groupIds
		response, err := client.TerminateInstances(request)
		if err != nil || response.Response == nil || response.Response.RequestId == nil {
			return nil, fmt.Errorf("terminate cvm instances: %w", err)
		}
	}

	return map[string]any{
		"success": true,
	}, nil
}

type CvmStopInstances struct {
	Region      string    `json:"region"`
	InstanceIds []*string `json:"instanceIds"`
}

func (self *CvmStopInstances) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/tencentcloud#cvmStopInstances")

	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return &CvmStopInstances{}
		},
		InputForm: spec.InputSchema,
	}
}

func (self *CvmStopInstances) Run(c *workflow.NodeContext) (any, error) {
	client, err := newCvmClient(c.Context(), self.Region, c.GetAuthorizer())
	if err != nil {
		return nil, fmt.Errorf("new cvm client: %w", err)
	}
	request := cvm.NewStopInstancesRequest()

	for _, groupIds := range collection.Chunk(self.InstanceIds, 100) {
		request.InstanceIds = groupIds
		response, err := client.StopInstances(request)
		if err != nil || response.Response == nil || response.Response.RequestId == nil {
			return nil, fmt.Errorf("terminate cvm instances: %w", err)
		}
	}

	return map[string]any{
		"success": true,
	}, nil
}

type tencentcloudCredentialMeta struct {
	SecretID  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
}

func newCvmClient(ctx context.Context, region string, decoder auth.CredentialMetaDecoder) (*cvm.Client, error) {
	meta := &tencentcloudCredentialMeta{}
	err := decoder.DecodeMeta(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to decode credential meta: %v", err)
	}
	credential := common.NewCredential(meta.SecretID, meta.SecretKey)
	client, err := cvm.NewClient(credential, region, profile.NewClientProfile())
	if err != nil {
		return nil, fmt.Errorf("new cvm client: %w", err)
	}
	return client, err
}
