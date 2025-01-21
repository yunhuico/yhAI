package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/port"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func (s *testServer) getSamples(workflowID, nodeID string, assertFns ...assertOptFn) []response.SampleData {
	opt := getAssertOpt(assertFns)

	resp := s.request("GET", "/api/v1/workflows/"+workflowID+"/nodes/"+nodeID+"/samples", nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.NodeSamplesResp{},
	}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)

	if opt.errorCode != 0 {
		return nil
	}

	return removeSampleNamesForCompare(r.Data.(*response.NodeSamplesResp).Samples)
}

func (s *testServer) loadMoreSamples(workflowID, nodeID string, assertFns ...assertOptFn) {
	opt := getAssertOpt(assertFns)

	resp := s.request("POST", "/api/v1/workflows/"+workflowID+"/nodes/"+nodeID+"/samples/loadMore", nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
}

func (s *testServer) selectSample(workflowID, nodeID string, sampleID int, assertFns ...assertOptFn) {
	opt := getAssertOpt(assertFns)
	resp := s.request("POST", "/api/v1/workflows/"+workflowID+"/nodes/"+nodeID+"/samples/"+strconv.Itoa(sampleID)+"/select", nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{}
	err := unmarshalResponse(resp, r)

	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
}

func (s *testServer) skipTest(workflowID, nodeID string, assertFns ...assertOptFn) {
	opt := getAssertOpt(assertFns)
	resp := s.request("POST", "/api/v1/workflows/"+workflowID+"/nodes/"+nodeID+"/skipTest", nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{}
	err := unmarshalResponse(resp, r)

	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
}

func (s *testServer) getAllNodeSamples(workflowID string, assertFns ...assertOptFn) *response.WorkflowSamplesResp {
	opt := getAssertOpt(assertFns)

	resp := s.request("GET", "/api/v1/workflows/"+workflowID+"/allNodeSamples", nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.WorkflowSamplesResp{},
	}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)

	if opt.errorCode != 0 {
		return nil
	}

	return r.Data.(*response.WorkflowSamplesResp)
}

// TestSamples
func TestSamplesWorksSuccessfully(t *testing.T) {
	assert := require.New(t)
	accessToken := "this is a test access token"
	freeport, err := port.GetFreePort()
	assert.NoError(err)
	if !startGitlabMockServer(t, freeport, accessToken) {
		return
	}

	server := newTestServer(t)

	credentialName := "gitlab access token"
	credentialID := server.createGitlabAccessTokenCredential(credentialName, fmt.Sprintf("http://localhost:%d", freeport), accessToken, false)

	workflowID := server.createWorkflow(&payload.EditWorkflowReq{
		Name: "workflow",
	})
	nodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:         "get issue list",
			Class:        "ultrafox/gitlab#triggerIssue",
			CredentialID: credentialID,
		},
		IsStart: true,
		InputFields: map[string]any{
			"projectId": "2",
			"event":     "all",
			"scope":     "project",
		},
	})

	// step1: get samples, no samples
	samples := server.getSamples(workflowID, nodeID)
	assert.Len(samples, 0)

	// step2: load samples
	server.loadMoreSamples(workflowID, nodeID)

	// step3: get 2 samples
	samples1 := server.getSamples(workflowID, nodeID)
	assert.Len(samples1, 2)
	samples2 := server.getSamples(workflowID, nodeID)
	assert.Len(samples2, 2)
	assert.Equal(samples1, samples2)
	assert.NotEmpty(samples1[0].RawOutput)
	assert.NotEmpty(samples1[0].FlattenOutput)

	request := workflow.HTTPRequest{
		Body: []byte(`{"object_attributes": {"iid": 1}}`),
	}
	b, _ := json.Marshal(request)
	// step4: run workflow
	server.runWorkflow(workflowID, &payload.RunWorkflowReq{
		NodeID:           nodeID,
		UseExternalInput: true,
		Input:            b,
	})

	// step5: still get 2 samples
	samples3 := server.getSamples(workflowID, nodeID)
	assert.Len(samples3, 2)

	// step6: load samples again
	server.loadMoreSamples(workflowID, nodeID)

	// step7: get 3 samples
	samples4 := server.getSamples(workflowID, nodeID)
	assert.Len(samples4, 3)
	assert.Equal(samples4[1:], samples3)

	// step8: select another sample
	server.selectSample(workflowID, nodeID, samples4[0].SampleID)
	samples5 := server.getSamples(workflowID, nodeID)
	assert.True(samples5[0].IsSelected)
	assert.False(samples5[1].IsSelected)
	assert.False(samples5[2].IsSelected)

	// step9: skip test
	server.skipTest(workflowID, nodeID)
	server.loadMoreSamples(workflowID, nodeID, assertErrorCode(codeNoMoreSamplesToLoad))

	// step10: get 4 samples because of the skip test.
	samples6 := server.getSamples(workflowID, nodeID)
	assert.Len(samples6, 4)
	assert.True(samples6[0].IsSelected)
	assert.Equal(model.NodeSourceSkip, samples6[0].Source)

	// step11: get all node samples
	allSamples := server.getAllNodeSamples(workflowID)
	assert.Len(allSamples.Samples, 1)
}

func removeSampleNamesForCompare(samples []response.SampleData) []response.SampleData {
	for i := range samples {
		samples[i].SampleName = ""
	}
	return samples
}

func TestSamplesExceptions(t *testing.T) {
	t.Run("when workflow not exists", func(t *testing.T) {
		server := newTestServer(t)
		server.getSamples("not_exists", "nodeID", assertErrorCode(httpbase.CodeGeneralError))
		server.loadMoreSamples("not_exists", "nodeID", assertErrorCode(httpbase.CodeGeneralError))
		server.selectSample("not_exists", "nodeID", 1, assertErrorCode(httpbase.CodeGeneralError))
		server.skipTest("not_exists", "nodeID", assertErrorCode(httpbase.CodeGeneralError))
	})

	t.Run("when node id not exists", func(t *testing.T) {
		server := newTestServer(t)
		workflowID := server.createWorkflow(&payload.EditWorkflowReq{
			Name: "workflow",
		})
		server.getSamples(workflowID, "nodeID", assertErrorCode(httpbase.CodeGeneralError))
		server.loadMoreSamples(workflowID, "nodeID", assertErrorCode(httpbase.CodeGeneralError))
		server.selectSample(workflowID, "nodeID", 1, assertErrorCode(httpbase.CodeGeneralError))
		server.skipTest(workflowID, "nodeID", assertErrorCode(httpbase.CodeGeneralError))
	})
}
