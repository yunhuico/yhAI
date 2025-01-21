package gitlab

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func TestListGroupEpic_QueryFieldResultList(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	log.Init("go-test", log.DebugLevel)
	client, err := gitlab.NewClient("token", gitlab.WithBaseURL("http://localhost:8180"), gitlab.WithHTTPClient(http.DefaultClient))
	assert.NoError(t, err)
	e := &ListGroupEpic{
		BaseGitlabNode: BaseGitlabNode{
			client: client,
		},
		GroupID: 1,
	}

	httpmock.RegisterResponder("GET", "http://localhost:8180/api/v4/groups/1/epics", func(request *http.Request) (*http.Response, error) {
		resp := httpmock.NewBytesResponse(200, []byte(`[
	{
		"id": 1000,
		"iid": 1,
		"title": "epic 1"
	},
	{
		"id": 2000,
		"iid": 2,
		"title": "epic 2"
	}
]`))
		return resp, nil
	})
	result, err := e.QueryFieldResultList(&workflow.NodeContext{
		TriggerNodeContext: workflow.NewTriggerNodeContext(context.Background()),
	})
	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, 1, result.Items[0].Value)
	assert.Equal(t, 2, result.Items[1].Value)
}
