package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func (s *testServer) getCredentials() []model.Credential {
	resp := s.request("GET", "/api/v1/credentials", nil)
	s.assertResponseOK(resp)

	r := &R{
		Data: &response.ListCredentialResp{},
	}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	return r.Data.(*response.ListCredentialResp).Credentials
}

func (s *testServer) getCredential(credentialID string) *response.GetCredentialResp {
	resp := s.request("GET", fmt.Sprintf("/api/v1/credentials/%s", credentialID), nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.GetCredentialResp{},
	}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	return r.Data.(*response.GetCredentialResp)
}

func (s *testServer) updateCredential(credentialID string, req *payload.EditCredentialReq, skipTesting bool) {
	b, err := json.Marshal(req)
	s.NoError(err)

	uri := fmt.Sprintf("/api/v1/credentials/%s", credentialID)
	if skipTesting {
		uri += fmt.Sprintf("?%s=1", skipTestCredentialKey)
	}
	resp := s.request("PUT", uri, bytes.NewReader(b))
	s.Equal(http.StatusOK, resp.Code)

	r := &R{}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
}

func (s *testServer) deleteCredential(credentialID string) {
	resp := s.request("DELETE", fmt.Sprintf("/api/v1/credentials/%s", credentialID), nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
}

func (s *testServer) getCredentialAssociatedWorkflows(credentialID string) []model.Workflow {
	resp := s.request("GET", fmt.Sprintf("/api/v1/credentials/%s/associatedWorkflows", credentialID), nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.ListAssociatedWorkflowsResp{},
	}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	return r.Data.(*response.ListAssociatedWorkflowsResp).Workflows
}

// TestCredentials
//
// step1: assert current credentials empty
// step2: create new credential
// step3: assert total 1 credentials
// step4: get the new credential
// step5: update token and name
// step6: update name only
// step7: delete credential
func TestCredentials(t *testing.T) {
	// os.Setenv("DB_DEBUG", "2")
	server := newTestServer(t)
	credentials := server.getCredentials()
	assert.Empty(t, credentials)

	credentialName := "gitlab access token"
	token := "!!!this-is-a-token!!!"
	credentialID := server.createGitlabAccessTokenCredential(credentialName, "https://gitlab.com", token, true)
	credentials = server.getCredentials()
	assert.Len(t, credentials, 1)
	assert.Equal(t, credentialID, credentials[0].ID)

	credential := server.getCredential(credentialID)
	assert.Equal(t, credentialName, credential.Name)
	assert.Equal(t, fmt.Sprintf("!!!%s!!!", strings.Repeat("*", len(token)-6)), credential.InputFields["accessToken"])

	newName := credentialName + "!!!"
	server.updateCredential(credentialID, &payload.EditCredentialReq{
		EditableCredential: model.EditableCredential{
			Name:         newName,
			AdapterClass: credential.AdapterClass,
			Type:         credential.Type,
		},
		InputFields: map[string]string{
			"server":      credential.InputFields["server"],
			"accessToken": token[1 : len(token)-1],
		},
	}, true)

	credential = server.getCredential(credentialID)
	assert.Equal(t, newName, credential.Name)
	assert.Equal(t, "!!t*************n!!", credential.InputFields["accessToken"])

	server.updateCredential(credentialID, &payload.EditCredentialReq{
		EditableCredential: model.EditableCredential{
			Name:         credentialName,
			AdapterClass: credential.AdapterClass,
			Type:         credential.Type,
		},
		InputFields: credential.InputFields,
	}, true)
	credential = server.getCredential(credentialID)
	assert.Equal(t, credentialName, credential.Name)
	assert.Equal(t, "!!t*************n!!", credential.InputFields["accessToken"])

	assert.Len(t, server.getCredentialAssociatedWorkflows(credentialID), 0)
	// create 10 enabled workflow(with trigger node)
	workflowIDs := [10]string{}
	nodeIDs := [10]string{}
	for i := 0; i < 10; i++ {
		workflow := &model.Workflow{
			OwnerRef: model.OwnerRef{
				OwnerType: model.OwnerTypeUser,
				OwnerID:   42,
			},
			Name:        fmt.Sprintf("workflow-%d", i),
			Status:      model.WorkflowStatusEnabled,
			StartNodeID: "triggerNodeID",
		}
		err := server.db.InsertWorkflow(ctx, workflow)
		assert.NoError(t, err)
		node := &model.Node{
			EditableNode: model.EditableNode{
				Name:         fmt.Sprintf("node-%d", i),
				Class:        "ultrafox/slack#triggerMessage",
				CredentialID: credentialID,
			},
			Type:       model.NodeTypeActor,
			WorkflowID: workflow.ID,
		}
		err = server.db.InsertNode(ctx, node)
		assert.NoError(t, err)
		workflowIDs[i] = workflow.ID
		nodeIDs[i] = node.ID
	}
	assert.Len(t, server.getCredentialAssociatedWorkflows(credentialID), 10)

	server.deleteCredential(credentialID)
	for _, workflowID := range workflowIDs {
		currentWorkflow, err := server.db.GetWorkflowByID(ctx, workflowID)
		assert.NoError(t, err)
		assert.Equal(t, model.WorkflowStatusDisabled, currentWorkflow.Status)
	}
	for _, nodeID := range nodeIDs {
		currentNode, err := server.db.GetNodeByID(ctx, nodeID)
		assert.NoError(t, err)
		assert.Equal(t, "", currentNode.CredentialID)
	}
}

func TestOfficialOAuth2Credentials(t *testing.T) {
	server := newTestServer(t)
	server.handler.officialCredentials = model.OfficialCredentials{
		{
			Name:    "Slack",
			Adapter: "ultrafox/slack",
			Type:    "oauth2",
		},
	}

	// step2: create a new credential from official credential
	credentialID := server.createCredential(&payload.EditCredentialReq{
		EditableCredential: model.EditableCredential{
			Name:         "slack",
			AdapterClass: "ultrafox/slack",
			Type:         model.CredentialTypeOAuth2,
			OfficialName: "Slack",
		},
	}, false)
	assert.NotEmpty(t, credentialID)

	// step3: get the created credential details
	credential := server.getCredential(credentialID)
	assert.NotNil(t, credential)
	assert.Equal(t, "Slack", credential.OfficialName)

	// step4: get current credentials
	credentials := server.getCredentials()
	assert.Len(t, credentials, 1)

	// step5: create a new credential from the wrong official credential
	{
		credential := &payload.EditCredentialReq{
			EditableCredential: model.EditableCredential{
				Name:         "Slack #1",
				AdapterClass: "ultrafox/slack",
				Type:         model.CredentialTypeOAuth2,
				OfficialName: "wrong-name",
			},
		}
		b, err := json.Marshal(credential)
		assert.NoError(t, err)
		resp := server.request("POST", "/api/v1/credentials", bytes.NewReader(b))
		r := &R{}
		err = unmarshalResponse(resp, r)
		assert.NoError(t, err)
		assert.NotEqual(t, 0, r.Code)
	}

	// step6: request a auth url
	result := server.requestAuthURL(t, credentialID, true)
	assert.NotEmpty(t, result.AuthURL)
	assert.True(t, strings.HasSuffix(result.AuthURL, result.StateID))
}

func TestCredentialValidateDynamically(t *testing.T) {
	server := newTestServer(t)
	server.createGitlabAccessTokenCredential(
		"gitlab accessToken",
		"https://gitlab.com",
		"",
		false, // need not skip testing credential, because verifying the required fields first, then testing credential.
		assertErrorCode(httpbase.CodeDynamicalFormInputError))

	credentialID := server.createGitlabAccessTokenCredential(
		"gitlab accessToken",
		"",
		"not_empty_token",
		false, // need not skip testing credential, because verifying the required fields first, then testing credential.
		assertErrorCode(httpbase.CodeDynamicalFormInputError))
	assert.Empty(t, credentialID)

	credentialID = server.createGitlabAccessTokenCredential(
		"gitlab accessToken",
		"https://jihulab.com",
		"not_empty_token",
		true, // should skip testing credential, because Verifying the required fields first, then testing credential.
	)
	assert.NotEmpty(t, credentialID)
}
