package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/port"
)

func (s *testServer) queryFieldSelect(req *payload.QueryFieldSelectReq) *response.QueryFieldSelectResp {
	b, err := json.Marshal(req)
	s.NoError(err)
	resp := s.request("POST", "/api/v1/adapters/fieldSelect", bytes.NewReader(b))
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.QueryFieldSelectResp{},
	}
	err = unmarshalResponse(resp, r)
	s.NoError(err)

	return r.Data.(*response.QueryFieldSelectResp)
}

func TestQueryFieldSelect_Basic(t *testing.T) {
	freeport, err := port.GetFreePort()
	assert.NoError(t, err)
	accessToken := "this is a test access token"

	if !startGitlabMockServer(t, freeport, accessToken) {
		return
	}

	server := newTestServer(t)
	credentialName := "gitlab access token"
	credentialID := server.createGitlabAccessTokenCredential(credentialName, fmt.Sprintf("http://localhost:%d", freeport), accessToken, false)

	t.Run("basic case, all values are static", func(t *testing.T) {
		assert := require.New(t)
		resp := server.queryFieldSelect(&payload.QueryFieldSelectReq{
			Class:        "ultrafox/gitlab#listProject",
			WorkflowID:   "",
			CredentialID: credentialID,
		})
		assert.Len(resp.Result.Items, 2)

		resp2 := server.queryFieldSelect(&payload.QueryFieldSelectReq{
			Class:        "ultrafox/gitlab#listProject",
			WorkflowID:   "",
			CredentialID: credentialID,
			PerPage:      1,
		})
		assert.Len(resp2.Result.Items, 1)

		resp3 := server.queryFieldSelect(&payload.QueryFieldSelectReq{
			Class:        "ultrafox/gitlab#listProject",
			WorkflowID:   "",
			CredentialID: credentialID,
			PerPage:      1,
			Page:         2,
		})
		assert.Len(resp2.Result.Items, 1)

		assert.NotEqual(resp2.Result.Items[0].Value, resp3.Result.Items[0].Value)
	})

	t.Run("basic case, values are dynamic", func(t *testing.T) {
		assert := require.New(t)
		startNodeID := "startNodeID"
		workflowID := "workflowID"

		resp := server.queryFieldSelect(&payload.QueryFieldSelectReq{
			Class:        "ultrafox/gitlab#listIssue",
			WorkflowID:   workflowID,
			CredentialID: credentialID,
			InputFields: map[string]any{
				"projectId": 1, // odd id
			},
			PerPage: 2,
			Page:    1,
		})
		assert.Len(resp.Result.Items, 0)
		assert.Equal(true, resp.Result.NoMore)

		// mock a workflow with a sample node
		err := server.db.InsertWorkflowInstanceNode(ctx, &model.WorkflowInstanceNode{
			WorkflowID:       workflowID,
			NodeID:           startNodeID,
			Class:            "ultrafox/gitlab#issueTrigger",
			Status:           model.WorkflowInstanceNodeStatusCompleted,
			Output:           []byte(`{"projectId": 2}`),
			StartTime:        time.Now(),
			Source:           model.NodeSourceLive,
			IsSelectedSample: true,
			IsSample:         true,
			SampleResourceID: "abc",
			SampledAt:        time.Now(),
			SampleVersion:    "abc",
		})
		assert.NoError(err)

		resp = server.queryFieldSelect(&payload.QueryFieldSelectReq{
			Class:        "ultrafox/gitlab#listIssue",
			WorkflowID:   workflowID,
			CredentialID: credentialID,
			InputFields: map[string]any{
				"projectId": fmt.Sprintf("{{ .Node.%s.output.projectId }}", startNodeID), // use a dynamic field
			},
			PerPage: 2,
			Page:    1,
		})
		assert.Len(resp.Result.Items, 2)
	})
}
