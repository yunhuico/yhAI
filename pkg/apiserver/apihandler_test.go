package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/gitlab"
	_ "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/schedule"
	_ "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/slack"
	_ "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/tencentcloud"

	"github.com/jarcoal/httpmock"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"
	xoauth2 "golang.org/x/oauth2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/port"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var ctx = context.Background()

func init() {
	log.Init("go-test.apiserver", log.DebugLevel)
}

type testServer struct {
	*require.Assertions

	serverHost *serverhost.ServerHost
	handler    *APIHandler
	router     *gin.Engine
	db         *model.DB
}

func newTestServer(t *testing.T) *testServer {
	assert := require.New(t)

	db, err := model.NewDB(ctx, model.DBConfig{
		Dialect: model.DialectSQLite,
		DSN:     "file::memory:?cache=shared",
	})
	assert.NoError(err)

	migrator := model.NewMigrator(db)
	err = migrator.Init(ctx)
	assert.NoError(err)
	_, err = migrator.Rollback(ctx)
	assert.NoError(err)
	_, err = migrator.Migrate(ctx)
	assert.NoError(err)
	err = db.InsertMockUserAndSession(ctx)
	assert.NoError(err)

	serverHostOpt := serverhost.Opt{
		API:     "http://localhost:8080",
		WebHook: "http://localhost:8081",
	}
	serverHost, err := serverhost.New(serverHostOpt)
	assert.NoError(err)

	cipher, err := crypto.NewASECipher(bytes.Repeat([]byte(`abcd`), 4))
	assert.NoError(err)

	passportVendors := model.PassportVendors{
		{
			Enabled:      true,
			Name:         "Jihulab",
			ClientID:     "client id here",
			ClientSecret: "client secret here",
			BaseURL:      "https://jihulab.com",
		},
		{
			Enabled:      true,
			Name:         "Gitlab",
			ClientID:     "client id here",
			ClientSecret: "client secret here",
			BaseURL:      "https://gitlab.com",
		},
	}
	passportVendorlookup, err := passportVendors.MapByVendorName()
	assert.NoError(err)

	triggerRegistry, err := trigger.NewRegistry(trigger.RegistryOpt{
		WebhookProviders: workflow.WebhookProviders(),
		Cipher:           cipher,
		ServerHost:       serverHost,
		DkronConfig: trigger.DkronConfig{
			DkronInternalHost:   "http://localhost:9999",
			WebhookInternalHost: "http://localhost:4444",
			JobTags:             map[string]string{"dc": "dc1:1"},
		},
		DB:                   db,
		PassportVendorLookup: passportVendorlookup,
	})
	assert.NoError(err)

	handler, err := newAPIHandler(APIHandlerOpt{
		DB:              db,
		Cipher:          cipher,
		TriggerRegistry: triggerRegistry,
		PassportVendors: passportVendors,
		ServerHost:      serverHost,
		BetaConfig: BetaConfig{
			InvitationSignUpSheetURL: "https://example.com",
			APIBearerToken:           "dummy",
		},
	})
	assert.NoError(err)

	middleware := &httpbase.Middleware{
		Logger: log.Clone(),
	}
	router := newGin(middleware)
	initAPIRouter(handler, router)

	return &testServer{
		Assertions: assert,
		handler:    handler,
		router:     router,
		db:         db,
		serverHost: serverHost,
	}
}

func (s *testServer) request(method, uri string, data io.Reader) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, uri, data)
	req.AddCookie(&http.Cookie{
		Name:     sessionCookieKey,
		Value:    "42",
		Expires:  time.Now().Add(time.Hour),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	s.router.ServeHTTP(w, req)
	return w
}

func (s *testServer) createGitlabAccessTokenCredential(name, server, accessToken string, skipTesting bool, fns ...assertOptFn) string {
	return s.createCredential(&payload.EditCredentialReq{
		EditableCredential: model.EditableCredential{
			Name:         name,
			AdapterClass: "ultrafox/gitlab",
			Type:         model.CredentialTypeAccessToken,
		},
		InputFields: map[string]string{
			"server":      server,
			"accessToken": accessToken,
		},
	}, skipTesting, fns...)
}

func (s *testServer) createCredential(credential *payload.EditCredentialReq, skipTesting bool, fns ...assertOptFn) string {
	opt := getAssertOpt(fns)
	b, err := json.Marshal(credential)
	s.NoError(err)
	uri := "/api/v1/credentials"
	if skipTesting {
		uri += fmt.Sprintf("?%s=1", skipTestCredentialKey)
	}
	resp := s.request("POST", uri, bytes.NewReader(b))
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.ResourceCreatedResponse{},
	}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
	if opt.errorCode != 0 {
		return ""
	}

	return r.Data.(*response.ResourceCreatedResponse).ID
}

func (s *testServer) createWorkflow(workflow *payload.EditWorkflowReq) string {
	b, err := json.Marshal(workflow)
	s.NoError(err)
	resp := s.request("POST", "/api/v1/workflows", bytes.NewReader(b))
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.ResourceCreatedResponse{},
	}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	return r.Data.(*response.ResourceCreatedResponse).ID
}

type assertOpt struct {
	errorCode      int
	httpStatusCode int
}
type assertOptFn func(*assertOpt)

func assertErrorCode(errCode int) assertOptFn {
	return func(opt *assertOpt) {
		opt.errorCode = errCode
	}
}

func assertHTTPStatusCode(code int) assertOptFn {
	return func(opt *assertOpt) {
		opt.httpStatusCode = code
	}
}

func getAssertOpt(fns []assertOptFn) *assertOpt {
	opt := &assertOpt{
		errorCode:      0,
		httpStatusCode: 200,
	}
	for _, f := range fns {
		f(opt)
	}
	return opt
}

func (s *testServer) getWorkflow(workflowID string, assertFns ...assertOptFn) *response.GetWorkflowResp {
	opt := getAssertOpt(assertFns)

	resp := s.request("GET", fmt.Sprintf("/api/v1/workflows/%s", workflowID), nil)

	s.Equal(http.StatusOK, resp.Code, fmt.Sprintf("body: %s", resp.Body.String()))
	r := &R{
		Data: &response.GetWorkflowResp{},
	}

	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
	if opt.errorCode != 0 {
		return nil
	}

	return r.Data.(*response.GetWorkflowResp)
}

func (s *testServer) deleteWorkflow(workflowID string, assertFns ...assertOptFn) {
	opt := getAssertOpt(assertFns)

	resp := s.request("DELETE", fmt.Sprintf("/api/v1/workflows/%s", workflowID), nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
}

func (s *testServer) createNode(workflowID string, p *payload.EditNodeReq, assertFns ...assertOptFn) string {
	opt := getAssertOpt(assertFns)

	b, err := json.Marshal(p)
	s.NoError(err)
	resp := s.request("POST", fmt.Sprintf("/api/v1/workflows/%s/nodes", workflowID), bytes.NewReader(b))
	s.assertResponseOK(resp)

	r := &R{
		Data: &response.ResourceCreatedResponse{},
	}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code, r.Msg)
	if opt.errorCode != 0 {
		return ""
	} else {
		return r.Data.(*response.ResourceCreatedResponse).ID
	}
}

func (s *testServer) updateNode(workflowID, nodeID string, p *payload.EditNodeReq, assertFns ...assertOptFn) {
	opt := getAssertOpt(assertFns)

	b, err := json.Marshal(p)
	s.NoError(err)
	resp := s.request("PUT", fmt.Sprintf("/api/v1/workflows/%s/nodes/%s", workflowID, nodeID), bytes.NewReader(b))
	s.assertResponseOK(resp)

	r := &R{}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
}

func (s *testServer) runNode(workflowID string, nodeID string, p *payload.RunNodeReq) *response.RunNodeResp {
	b, err := json.Marshal(p)
	s.NoError(err)
	resp := s.request("POST", fmt.Sprintf("/api/v1/workflows/%s/nodes/%s/run", workflowID, nodeID), bytes.NewReader(b))
	s.assertResponseOK(resp)

	r := &R{
		Data: &response.RunNodeResp{},
	}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(0, r.Code)

	return r.Data.(*response.RunNodeResp)
}

func assertResponseOK(t *testing.T, resp *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusOK, resp.Code, fmt.Sprintf("body: %s", resp.Body.String()))
}

func (s *testServer) updateNodeTransition(workflowID, nodeID string, p *payload.UpdateNodeTransitionReq) {
	b, err := json.Marshal(p)
	s.NoError(err)
	resp := s.request("PUT", fmt.Sprintf("/api/v1/workflows/%s/nodes/%s/transition", workflowID, nodeID), bytes.NewReader(b))
	s.assertResponseOK(resp)

	r := &R{}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(0, r.Code)
}

func (s *testServer) enableWorkflow(workflowID string) {
	resp := s.request("POST", fmt.Sprintf("/api/v1/workflows/%s/enable", workflowID), nil)
	s.assertResponseOK(resp)

	r := &R{}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(0, r.Code, r.Msg)
}

func (s *testServer) disableWorkflow(workflowID string) {
	resp := s.request("POST", fmt.Sprintf("/api/v1/workflows/%s/disable", workflowID), nil)
	s.assertResponseOK(resp)

	r := &R{}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(0, r.Code)
}

func (s *testServer) ListOfficialOAuth2Credentials(t *testing.T) *response.ListOfficialCredentialsResp {
	resp := s.request("GET", "/api/v1/credentials/oauth2/official", nil)
	assertResponseOK(t, resp)

	r := &R{
		Data: &response.ListOfficialCredentialsResp{},
	}
	err := unmarshalResponse(resp, r)
	assert.NoError(t, err)
	assert.Equal(t, 0, r.Code)
	return r.Data.(*response.ListOfficialCredentialsResp)
}

func (s *testServer) requestAuthURL(t *testing.T, id string, force bool) *response.RequestAuthURLResponse {
	req := payload.RequestAuthURLReq{
		CredentialID: id,
		ForceRefresh: force,
	}
	b, err := json.Marshal(req)
	assert.NoError(t, err)
	resp := s.request("POST", "/api/v1/credentials/oauth2/authUrl", bytes.NewReader(b))
	assert.Equal(t, http.StatusOK, resp.Code)

	r := &R{
		Data: &response.RequestAuthURLResponse{},
	}
	err = unmarshalResponse(resp, r)
	assert.NoError(t, err)
	return r.Data.(*response.RequestAuthURLResponse)
}

func (s *testServer) oauth2Callback(t *testing.T, state, code string) {
	resp := s.request("GET", fmt.Sprintf("/api/v1/credentials/oauth2/callback?state=%s&code=%s", state, code), nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	r := &R{}
	err := unmarshalResponse(resp, r)
	assert.NoError(t, err)
}

func (s *testServer) runWorkflow(id string, p *payload.RunWorkflowReq) {
	b, err := json.Marshal(p)
	s.NoError(err)
	resp := s.request("POST", fmt.Sprintf("/api/v1/workflows/%s/run", id), bytes.NewReader(b))
	s.assertResponseOK(resp)

	r := &R{}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(0, r.Code)
}

func (s *testServer) assertResponseOK(resp *httptest.ResponseRecorder) {
	s.Equal(http.StatusOK, resp.Code, fmt.Sprintf("body: %s", resp.Body.String()))
}

func (s *testServer) serverMeta() *response.ServerMetaResp {
	resp := s.request("GET", "/api/v1/server/meta", nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &response.ServerMetaResp{},
	}
	err := unmarshalResponse(resp, r)
	s.NoError(err)

	return r.Data.(*response.ServerMetaResp)
}

func (s *testServer) deleteNode(workflowID, nodeID string, assertFns ...assertOptFn) {
	opt := getAssertOpt(assertFns)
	resp := s.request("DELETE", fmt.Sprintf("/api/v1/workflows/%s/nodes/%s", workflowID, nodeID), nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{}
	err := unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
}

func (s *testServer) listAdapters() *adapter.ListPresentData {
	resp := s.request("GET", "/api/v1/adapters", nil)
	s.Equal(http.StatusOK, resp.Code)

	r := &R{
		Data: &adapter.ListPresentData{},
	}
	err := unmarshalResponse(resp, r)
	s.NoError(err)

	return r.Data.(*adapter.ListPresentData)
}

func unmarshalResponse(response *httptest.ResponseRecorder, obj any) error {
	return json.Unmarshal(response.Body.Bytes(), obj)
}

type mockGitlabServer struct {
	server *http.Server
	engine *gin.Engine
}

func newMockGitlabServer(port int, token string) *mockGitlabServer {
	engine := gin.Default()
	engine.GET("/api/v4/projects/:id/issues", func(c *gin.Context) {
		// odd number id return empty
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if id%2 == 1 {
			c.Data(200, "application/json", []byte(`[]`))
			return
		}

		if c.GetHeader("PRIVATE-TOKEN") != token {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Data(200, "application/json", mustReadFile("./testdata/issueList.json"))
	})

	engine.GET("/api/v4/user", func(c *gin.Context) {
		c.JSON(200, &gitlab.User{
			ID:        1,
			Name:      "Ultrafox",
			Username:  "Ultrafox",
			AvatarURL: "https://ultrafox.dev",
			Email:     "ultrafox@ultrafox.com",
		})
	})

	engine.GET("/api/v4/projects/:id", func(c *gin.Context) {
		c.JSON(200, &gitlab.Project{
			ID:            1,
			Name:          "Ultrafox",
			Description:   "Automate everything.",
			AvatarURL:     "https://ultrafox.dev",
			SSHURLToRepo:  "git@jihulab.com:ultrafox/ultrafox.git",
			HTTPURLToRepo: "https://jihulab.com/ultrafox/ultrafox.git",
			Namespace: &gitlab.ProjectNamespace{
				ID:   1,
				Name: "Ultrafox Project",
				Path: "ultrafox",
			},
			PathWithNamespace: "ultrafox/ultrafox",
			DefaultBranch:     "main",
			WebURL:            "https://jihulab.com/ultrafox/ultrafox",
		})
	})

	engine.GET("/api/v4/projects", func(c *gin.Context) {
		if c.GetHeader("PRIVATE-TOKEN") != token {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var (
			page, _    = strconv.Atoi(c.Query("page"))
			perPage, _ = strconv.Atoi(c.Query("per_page"))
		)
		if page <= 0 {
			page = 1
		}
		if perPage <= 0 {
			perPage = 100
		}

		start := (page - 1) * perPage
		end := page * perPage

		b := mustReadFile("./testdata/projectList.json")
		var list []any
		_ = json.Unmarshal(b, &list)

		if start >= len(list) {
			c.JSON(200, nil)
		} else if end >= len(list) {
			c.JSON(200, list[start:])
		} else {
			c.JSON(200, list[start:end])
		}
	})

	engine.POST("/oauth/token", func(c *gin.Context) {
		c.JSON(200, xoauth2.Token{
			AccessToken:  token,
			TokenType:    "bearer",
			RefreshToken: "refreshToken",
			Expiry:       time.Now().Add(time.Hour),
		})
	})

	engine.GET("/api/v4/version", func(c *gin.Context) {
		c.JSON(200, map[string]string{
			"version": "latest",
		})
	})

	return &mockGitlabServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: engine,
		},
	}
}

func (s *mockGitlabServer) start() error {
	return s.server.ListenAndServe()
}

func mustReadFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

// TestBuildWorkflow
// step1: create a workflow
// step2: create a accessToken credential
// step3: create a start node (get issue list)
// step4: create a debug node (debug issue length)
// step5: update start node transition to debug node
// step6: enable the workflow
// step7: run the start node
// step8: run the second node
// step9: create a foreach node
// step10: create a foreach-sub1 node (switch issue id is odd)
// step11: create a foreach-sub2 node (just debug something)
// step12: test foreach node itself (https://jihulab.com/ultrafox/ultrafox/-/issues/513)
// step13: run foreach-sub1 node
// step14: run foreach-sub2 node
// step15: run the workflow
// step16: disable the workflow
// step17: delete the node
func TestBuildWorkflow(t *testing.T) {
	assert := require.New(t)
	freeport, err := port.GetFreePort()
	assert.NoError(err)
	accessToken := "this is a test access token"

	if !startGitlabMockServer(t, freeport, accessToken) {
		return
	}

	server := newTestServer(t)
	workflowName := "this is a workflow"
	workflowID := server.createWorkflow(&payload.EditWorkflowReq{
		Name: workflowName,
	})
	workflow := server.getWorkflow(workflowID) // step1
	assert.Equal(workflowName, workflow.Workflow.Name)

	credentialName := "gitlab access token"
	credentialID := server.createGitlabAccessTokenCredential(credentialName, fmt.Sprintf("http://localhost:%d", freeport), accessToken, false)
	assert.NotEmpty(credentialID)

	firstNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "Debug trigger",
			Class: "ultrafox/debug#triggerEcho",
		},
		IsStart: true,
	})

	getIssueListNodeID := server.createNode(workflowID, &payload.EditNodeReq{ // step3
		EditableNode: model.EditableNode{
			Name:         "get issue list",
			Class:        "ultrafox/gitlab#listIssue",
			CredentialID: credentialID,
		},
		IsStart: false,
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID: firstNodeID,
		},
		InputFields: map[string]any{
			"projectId": 2,
			"labels":    []string{},
		},
	})
	debugNodeID := server.createNode(workflowID, &payload.EditNodeReq{ // step4
		EditableNode: model.EditableNode{
			Name:  "print issue length",
			Class: "ultrafox/debug#printTarget",
		},
		IsStart: false,
		InputFields: map[string]any{
			"target": fmt.Sprintf("{{ .Node.%s.output | len }}", getIssueListNodeID),
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID: getIssueListNodeID,
		},
	})

	server.enableWorkflow(workflowID) // step6

	startOutput := server.runNode(workflowID, getIssueListNodeID, &payload.RunNodeReq{ // step7
	})
	assert.NotEmpty(startOutput.FlattenOutput)
	assert.Greater(len(startOutput.FlattenOutput[0].Fields), 3, "ultrafox/gitlab#listIssue foreach field should have 3 sub fields")
	assert.Equal("backend", startOutput.FlattenOutput[0].Fields[2].AsStr, "ultrafox/gitlab#listIssue foreach field sub field [labels] asStr is 'backend'")
	assert.Equal(`.Iter.loopItem.labels`, startOutput.FlattenOutput[0].Fields[2].Reference)

	debugOutput := server.runNode(workflowID, debugNodeID, &payload.RunNodeReq{ // step8
	})
	assert.Equal("2", debugOutput.RawOutput)

	// step9: create foreach nodes
	getForeachNodePayload := func(transition string) *payload.EditNodeReq {
		return &payload.EditNodeReq{
			EditableNode: model.EditableNode{
				Name:       "foreach all issues",
				Class:      "ultrafox/foreach#loopFromList",
				Transition: "",
			},
			InputFields: map[string]any{
				"inputCollection": startOutput.FlattenOutput[0].Reference,
				"transition":      transition,
			},
			PreviousNodeInfo: payload.PreviousNodeInfo{
				PreviousNodeID: debugNodeID,
			},
		}
	}
	foreachNodeID := server.createNode(workflowID, getForeachNodePayload(""))

	getSwitchPayload := func(transition string) *payload.EditNodeReq {
		return &payload.EditNodeReq{
			EditableNode: model.EditableNode{
				Name:  "switch issue id is odd",
				Class: "ultrafox/logic#switch",
			},
			InputFields: map[string]any{
				"paths": []any{
					map[string]any{
						"name": "is odd",
						"conditions": []any{
							[]any{
								map[string]any{
									"left":      `{{ mod .Iter.loopItem.id 2 }}`,
									"right":     "1",
									"operation": "equals",
								},
							},
						},
						"transition": transition,
					},
					map[string]any{
						"name":      "default",
						"isDefault": true,
					},
				},
			},
			PreviousNodeInfo: payload.PreviousNodeInfo{
				PreviousNodeID:    foreachNodeID,
				IsFirstInsideNode: true,
			},
		}
	}
	// step10: create foreach-sub node1
	foreachSwitchNodeID := server.createNode(workflowID, getSwitchPayload(""))

	// step11: create foreach-sub node2
	foreachDebugNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "just print issue title",
			Class: "ultrafox/debug#printTarget",
		},
		IsStart: false,
		InputFields: map[string]any{
			"target": "{{ .Iter.loopItem.title }}, assignees: {{ .Iter.loopItem.assignees[]?.username }}",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID:          foreachSwitchNodeID,
			PreviousSwitchPathIndex: 0,
			IsFirstInsideNode:       true,
		},
	})

	// step12: test foreach node itself
	foreachOutput := server.runNode(workflowID, foreachNodeID, nil)
	foreachOutputRawOutput := foreachOutput.RawOutput.(map[string]any)
	assert.Equal(float64(1), foreachOutputRawOutput["loopIteration"])
	assert.Equal(false, foreachOutputRawOutput["loopIterationIsLast"])
	assert.Equal(float64(2), foreachOutputRawOutput["loopTotalIterations"])
	assert.NotNil(foreachOutputRawOutput["loopItem"])
	assert.Nil(foreachOutputRawOutput["results"])
	assert.Equal(".Iter.loopTotalIterations", foreachOutput.FlattenOutput[0].Reference)
	assert.Equal("loopTotalIterations", foreachOutput.FlattenOutput[0].Key)
	assert.Equal("2", foreachOutput.FlattenOutput[0].AsStr)
	assert.Equal(".Iter.loopIteration", foreachOutput.FlattenOutput[1].Reference)
	assert.Equal(".Iter.loopIterationIsLast", foreachOutput.FlattenOutput[2].Reference)
	assert.Greater(len(foreachOutput.FlattenOutput), 3)
	for i := 3; i < len(foreachOutput.FlattenOutput); i++ {
		assert.True(strings.HasPrefix(foreachOutput.FlattenOutput[i].Reference, ".Iter.loopItem"))
	}

	foreachNodeSamples := server.getSamples(workflowID, foreachNodeID)
	assert.Len(foreachNodeSamples, 1)
	assert.Equal(".Iter.loopTotalIterations", foreachNodeSamples[0].FlattenOutput[0].Reference)
	assert.Equal("loopTotalIterations", foreachNodeSamples[0].FlattenOutput[0].Key)
	assert.Equal(".Iter.loopIteration", foreachNodeSamples[0].FlattenOutput[1].Reference)
	assert.Equal(".Iter.loopIterationIsLast", foreachNodeSamples[0].FlattenOutput[2].Reference)
	assert.Greater(len(foreachNodeSamples[0].FlattenOutput), 3)
	for i := 3; i < len(foreachNodeSamples[0].FlattenOutput); i++ {
		assert.True(strings.HasPrefix(foreachNodeSamples[0].FlattenOutput[i].Reference, ".Iter.loopItem"))
	}

	// step13: run foreach node1
	foreachSwitchNodeOutput := server.runNode(workflowID, foreachSwitchNodeID, &payload.RunNodeReq{
		ParentNodeID: foreachNodeID,
		IterIndex:    0,
	})
	assert.Equal([]any{
		map[string]any{"id": "1", "name": "is odd", "executionResult": true},
		map[string]any{"executionResult": false, "id": "default", "name": "default"}}, foreachSwitchNodeOutput.RawOutput.([]any))
	switchNodeSamples := server.getSamples(workflowID, foreachSwitchNodeID)
	assert.Len(switchNodeSamples, 1)

	foreachSwitchNodeOutput = server.runNode(workflowID, foreachSwitchNodeID, &payload.RunNodeReq{
		ParentNodeID: foreachNodeID,
		IterIndex:    1, // diff: the issue[1] id is even.
	})
	assert.Equal([]any{
		map[string]any{"id": "1", "name": "is odd", "executionResult": false},
		map[string]any{"id": "default", "name": "default", "executionResult": true}}, foreachSwitchNodeOutput.RawOutput.([]any))

	// actually there are two samples in database, but api just returns the selected sample.
	switchNodeSamples = server.getSamples(workflowID, foreachSwitchNodeID)
	assert.Len(switchNodeSamples, 1)
	count, err := server.db.CountSamplesByNodeID(ctx, foreachSwitchNodeID)
	assert.NoError(err)
	assert.Equal(2, count)

	// step14: run foreach node2
	foreachDebugNodeOutput := server.runNode(workflowID, foreachDebugNodeID, &payload.RunNodeReq{
		ParentNodeID: foreachNodeID,
		IterIndex:    0,
	})
	assert.Equal("add adapter output schema, assignees: jihulab,ultrafox", foreachDebugNodeOutput.RawOutput.(string))

	foreachDebugNodeOutput = server.runNode(workflowID, foreachDebugNodeID, &payload.RunNodeReq{
		ParentNodeID: foreachNodeID,
		IterIndex:    1,
	})
	assert.Equal("add e2e test, assignees: ", foreachDebugNodeOutput.RawOutput.(string))

	// step15: run the workflow
	server.runWorkflow(workflowID, &payload.RunWorkflowReq{
		NodeID: getIssueListNodeID,
	})
	instances, err := server.db.GetWorkflowInstancesByWorkflowID(ctx, workflowID)
	assert.NoError(err)
	assert.Len(instances, 1)
	assert.Equal(model.WorkflowInstanceStatusCompleted, instances[0].Status)

	allSamples := server.getAllNodeSamples(workflowID)
	assert.Len(allSamples.Samples, 5)
	assert.Contains(allSamples.Samples, getIssueListNodeID)
	assert.Contains(allSamples.Samples, debugNodeID)
	assert.Contains(allSamples.Samples, foreachDebugNodeID)
	assert.Contains(allSamples.Samples, foreachSwitchNodeID)
	assert.Contains(allSamples.Samples, foreachNodeID)
	assert.Equal(".Iter.loopTotalIterations", allSamples.Samples[foreachNodeID].FlattenOutput[0].Reference)
	assert.Equal("loopTotalIterations", allSamples.Samples[foreachNodeID].FlattenOutput[0].Key)
	assert.Equal(".Iter.loopIteration", allSamples.Samples[foreachNodeID].FlattenOutput[1].Reference)
	assert.Equal(".Iter.loopIterationIsLast", allSamples.Samples[foreachNodeID].FlattenOutput[2].Reference)
	assert.Greater(len(allSamples.Samples[foreachNodeID].FlattenOutput), 3)
	assert.Greater(len(allSamples.Samples[foreachNodeID].FlattenOutput), 3)
	for i := 3; i < len(allSamples.Samples[foreachNodeID].FlattenOutput); i++ {
		assert.True(strings.HasPrefix(foreachOutput.FlattenOutput[i].Reference, ".Iter.loopItem"))
	}

	server.disableWorkflow(workflowID) // step16

	server.deleteWorkflow(workflowID)
	server.getWorkflow(workflowID, assertErrorCode(httpbase.CodeGeneralError))
}

func TestForeachUseNotListDataAsInput(t *testing.T) {
	server := newTestServer(t)
	workflowName := "this is a workflow"
	workflowID := server.createWorkflow(&payload.EditWorkflowReq{
		Name: workflowName,
	})
	triggerNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "Debug trigger",
			Class: "ultrafox/schedule#cron",
		},
		InputFields: map[string]any{
			"expr":     "* * * * * *",
			"timezone": "UTC",
		},
		IsStart: true,
	})
	triggerNode, err := server.db.GetNodeByID(ctx, triggerNodeID)
	assert.NoError(t, err)
	assert.Equal(t, model.NodeTestingDefaultStatus, triggerNode.TestingStatus)
	foreachNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:       "foreach all issues",
			Class:      "ultrafox/foreach#loopFromList",
			Transition: "",
		},
		InputFields: map[string]any{
			"inputCollection": fmt.Sprintf(".Node.%s.output", triggerNodeID),
			"transition":      "",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID: triggerNodeID,
		},
	})
	foreachInternalNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "just print",
			Class: "ultrafox/debug#printTarget",
		},
		IsStart: false,
		InputFields: map[string]any{
			"target": "{{ .Iter.loopItem.datetime }}",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID:          foreachNodeID,
			PreviousSwitchPathIndex: 0,
			IsFirstInsideNode:       true,
		},
	})

	resp1 := server.runNode(workflowID, triggerNodeID, nil)
	triggerNode, err = server.db.GetNodeByID(ctx, triggerNodeID)
	assert.NoError(t, err)
	assert.Equal(t, model.NodeTestingSuccessStatus, triggerNode.TestingStatus)
	resp2 := server.runNode(workflowID, foreachNodeID, nil)
	resp3 := server.runNode(workflowID, foreachInternalNodeID, &payload.RunNodeReq{
		ParentNodeID: foreachNodeID,
	})
	foreachInternalNode, err := server.db.GetNodeByID(ctx, foreachInternalNodeID)
	assert.NoError(t, err)
	assert.Equal(t, model.NodeTestingSuccessStatus, foreachInternalNode.TestingStatus)
	// resp2 loopItem.datetime == resp1 datetime
	assert.Equal(t,
		resp1.RawOutput.(map[string]any)["datetime"],
		resp2.RawOutput.(map[string]any)["loopItem"].(map[string]any)["datetime"])
	// resp3 print datetime
	assert.Equal(t,
		resp1.RawOutput.(map[string]any)["datetime"],
		resp3.RawOutput)

	// update foreachInternalNode's inputFields, then testingStatus will become default.
	server.updateNode(workflowID, foreachInternalNodeID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "just print",
			Class: "ultrafox/debug#printTarget",
		},
		IsStart: false,
		InputFields: map[string]any{
			"target": "{{ .Iter.loopItem.datetime }}!!!",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID:          foreachNodeID,
			PreviousSwitchPathIndex: 0,
			IsFirstInsideNode:       true,
		},
	})
	foreachInternalNode, err = server.db.GetNodeByID(ctx, foreachInternalNodeID)
	assert.NoError(t, err)
	assert.Equal(t, model.NodeTestingDefaultStatus, foreachInternalNode.TestingStatus)
}

func startGitlabMockServer(t *testing.T, freeport int, token string) (success bool) {
	// mock a gitlab server
	mockGitlabServer := newMockGitlabServer(freeport, token)
	go func() {
		err := mockGitlabServer.start()
		assert.NoError(t, err)
	}()
	err := port.WaitPort(freeport, 2*time.Second)
	success = err == nil
	return
}

func TestServerMeta(t *testing.T) {
	server := newTestServer(t)
	meta := server.serverMeta()
	assert.NotNil(t, meta)
}

func TestEditWorkflow(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// mock Dkron response
	httpmock.RegisterResponder("POST", "http://localhost:9999/v1/jobs", httpmock.NewStringResponder(200, `{}`))

	server := newTestServer(t)
	workflowName := "this is a test workflow"
	workflowID := server.createWorkflow(&payload.EditWorkflowReq{
		Name: workflowName,
	})
	startNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "start node",
			Class: validate.CronTriggerClass,
		},
		IsStart: true,
		InputFields: map[string]any{
			"expr":     "* * * * * *",
			"timezone": "Asia/Shanghai",
		},
	})

	server.enableWorkflow(workflowID)
	t.Run("test delete trigger node fail when workflow enabled", func(t *testing.T) {
		server.deleteNode(workflowID, startNodeID, assertErrorCode(codeInternalError))
	})
	t.Run("test update trigger node fail when workflow disabled", func(t *testing.T) {
		server.updateNode(workflowID, startNodeID, &payload.EditNodeReq{
			EditableNode: model.EditableNode{},
			IsStart:      true,
			InputFields:  nil,
		}, assertErrorCode(codeInternalError))
	})
	t.Run("test delete workflow fail when workflow is enabled", func(t *testing.T) {
		server.deleteWorkflow(workflowID, assertErrorCode(codeInternalError))
	})
}

func TestListAdapters(t *testing.T) {
	server := newTestServer(t)
	adapters1 := server.listAdapters()
	adapterCount1 := len(adapters1.AdapterList)
	assert.Greater(t, adapterCount1, 0)

	adapterManager := adapter.GetAdapterManager()
	for _, meta := range adapterManager.GetMetas() {
		if meta.Class == adapters1.AdapterList[0].Class {
			for _, spec := range meta.Specs {
				spec.Hidden = true
			}
		}
	}

	adapters2 := server.listAdapters()
	adapterCount2 := len(adapters2.AdapterList)
	assert.Equal(t, adapterCount2+1, adapterCount1)
}

func TestBetaConfig_InvitationSignUpSheetURLWithEmail(t *testing.T) {
	tests := []struct {
		Name                              string
		Email                             string
		InvitationSignUpSheetURL          string
		InvitationSignUpSheetEmailFieldID string
		Want                              string
	}{
		{
			Name:                              "behavior before config updated",
			Email:                             "someone@example.com",
			InvitationSignUpSheetURL:          "https://example.com/sign-up",
			InvitationSignUpSheetEmailFieldID: "",
			Want:                              "https://example.com/sign-up",
		},
		{
			Name:                              "empty config",
			Email:                             "someone@example.com",
			InvitationSignUpSheetURL:          "",
			InvitationSignUpSheetEmailFieldID: "",
			Want:                              "",
		},
		{
			Name:                              "invalid sheet URL",
			Email:                             "someone@example.com",
			InvitationSignUpSheetURL:          "invalid",
			InvitationSignUpSheetEmailFieldID: "aaaa",
			Want:                              "invalid",
		},
		{
			Name:                              "current behavior",
			Email:                             "someone1234@example.com",
			InvitationSignUpSheetURL:          "https://example.com/sign-up",
			InvitationSignUpSheetEmailFieldID: "QWERTY123456",
			Want:                              "https://example.com/sign-up?QWERTY123456=someone1234%40example.com",
		},
		{
			Name:                              "current behavior with escaping",
			Email:                             "someone1234@example.com",
			InvitationSignUpSheetURL:          "https://example.com/sign-up",
			InvitationSignUpSheetEmailFieldID: "QWERTY123456@#$",
			Want:                              "https://example.com/sign-up?QWERTY123456%40%23%24=someone1234%40example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			c := BetaConfig{
				InvitationSignUpSheetURL:          tt.InvitationSignUpSheetURL,
				InvitationSignUpSheetEmailFieldID: tt.InvitationSignUpSheetEmailFieldID,
			}
			assert.Equalf(t, tt.Want, c.InvitationSignUpSheetURLWithEmail(tt.Email), "InvitationSignUpSheetURLWithEmail(%v)", tt.Email)
		})
	}
}
