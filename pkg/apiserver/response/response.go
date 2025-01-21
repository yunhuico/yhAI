package response

import (
	"encoding/json"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/schema"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/share"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
)

// ResourceCreatedResponse common response for resource created
type ResourceCreatedResponse struct {
	ID string `json:"id"`
}

type ListWorkflowResponse struct {
	Total     int                  `json:"total"`
	Workflows []*WorkflowWithIcons `json:"workflows"`
}

type WorkflowWithIcons struct {
	model.Workflow

	Icons []string `json:"icons"`
}

type ListWorkflowLogResp struct {
	Total             int                            `json:"total"`
	WorkflowInstances []WorkflowInstanceWithWorkflow `json:"workflowInstances"`
}

type WorkflowInstanceWithWorkflow struct {
	model.WorkflowInstance

	Workflow *model.Workflow `json:"workflow,omitempty"`
	Icons    []string        `json:"icons"`
}

type ListCredentialResp struct {
	Total       int                `json:"total"`
	Credentials []model.Credential `json:"credentials"`
}

type LoginStatusResp struct {
	SignedIn bool        `json:"signedIn"`
	User     *model.User `json:"user,omitempty"`
}

type GetCredentialResp struct {
	ID string `json:"id"`

	model.EditableCredential
	Status model.CredentialStatus `json:"status" bun:",notnull"`

	InputFields map[string]string `json:"inputFields"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type WorkflowDetail struct {
	*model.WorkflowWithNodes

	AllNodesPassTest bool `json:"allNodesPassTest"`
}

type GetWorkflowResp struct {
	Workflow WorkflowDetail `json:"workflow"`
}

type GetWorkflowExtraResp struct {
	AllNodesPassTest bool `json:"allNodesPassTest"`
}

type RunNodeResp struct {
	// RawOutput is raw json from adapter call result.
	RawOutput any `json:"rawOutput"`

	// FlattenOutput just for user select field.
	// DEPRECATED: will be removed.
	FlattenOutput []schema.OutputField `json:"flattenOutput"` // nolint: staticcheck
}

type GetOrgByIDResp struct {
	Organization model.Organization      `json:"organization"`
	Role         string                  `json:"role"`
	Permissions  []permission.Permission `json:"permissions"`
}

type ListOrgMembersResp struct {
	Total int                      `json:"total"`
	Users []model.OrganizationUser `json:"users"`
}

type BrowseOrgInviteResp struct {
	Organization model.Organization `json:"organization"`
}

type ListOfficialCredentialsResp struct {
	Credentials model.OfficialCredentials `json:"credentials"`
}

type RequestAuthURLResponse struct {
	StateID string `json:"stateId"`
	AuthURL string `json:"authUrl"`
}

type BrowseWorkflowShareLinkResp struct {
	model.WorkflowWithNodes

	Icons       []string          `json:"icons"`
	Annotations share.Annotations `json:"annotations"`
}

type ExportWorkflowResp struct {
	WorkflowYaml string `json:"workflowYaml"`
}

type WorkflowStatistics struct {
	Total                 int     `json:"total"`
	EnableCount           int     `json:"enableCount"`
	DisableCount          int     `json:"disableCount"`
	ExecutionCount        int     `json:"executionCount"`
	SuccessExecutionCount int     `json:"successExecutionCount"`
	FailExecutionCount    int     `json:"fail"`
	DayExecutionCount     [24]int `json:"dayExecution"`

	OrgStatistics model.WorkflowOrgStatistics      `json:"orgStatistics"`
	RunStatistics model.WorkflowInstanceStatistics `json:"runStatistics"`
}

type AdapterStatistics struct {
	UsageLeaderboard model.AdapterUsageStatistics `json:"adapterLeaderboard"`
	ActorCount       int                          `json:"actorCount"`
	TriggerCount     int                          `json:"triggerCount"`
}

type AdapterUsage struct {
	// Name of the adapter
	Name                 string `json:"name"`
	RelatedWorkflowCount int    `json:"relatedWorkflowCount"`
}

type StatisticsResp struct {
	Workflow WorkflowStatistics `json:"workflow"`
	Adapter  AdapterStatistics  `json:"adapter"`
	Org      OrgStatistics      `json:"org"`
}

type QueryFieldSelectResp struct {
	Result workflow.QueryFieldResult `json:"result"`
}

type ServerMetaResp struct {
	DocsHost          string `json:"docsHost"`
	OAuth2CallbackURL string `json:"oauth2CallbackUrl"`
}

type QueryAuthStateResponse struct {
	// Status 0:init, 1:completed, 2:failed
	Status model.OAuth2Status `json:"status"`
}

type OrgStatistics struct {
	Total int `json:"total"`
}

type SampleData struct {
	SampleID   int              `json:"sampleId"`
	SampleName string           `json:"sampleName"`
	IsSelected bool             `json:"isSelected"`
	SampledAt  time.Time        `json:"sampledAt"`
	Source     model.NodeSource `json:"source"`

	RunNodeResp
}

type NodeSamplesResp struct {
	Samples []SampleData `json:"samples"`
}

type WorkflowSamplesResp struct {
	Samples map[string]SampleData `json:"samples"`
}

type ListAssociatedWorkflowsResp struct {
	Workflows []model.Workflow `json:"workflows"`
}

type WorkflowInstanceNodeData struct {
	NodeID   string `json:"nodeId"`
	Input    any    `json:"input"`
	Output   any    `json:"output"`
	NodeName string `json:"nodeName"`
	Error    string `json:"error"`
}

type DetailedWorkflowInstanceResp struct {
	ID string `json:"id" bun:",pk"`

	WorkflowID  string                       `json:"workflowId"`
	Status      model.WorkflowInstanceStatus `json:"status"`
	StartNodeID string                       `json:"startNodeId"`

	// the id of failing node
	FailNodeID string    `json:"failNodeId"`
	StartTime  time.Time `json:"startTime"`
	// the length of execution, in milliseconds
	DurationMs int `json:"durationMs" bun:",nullzero"`
	// Data is sorted list (follow the order of the workflow diagram)
	Data []WorkflowInstanceNodeData `json:"data"`
}

// GetDetailedWorkflowInstanceResp gets the detailed workflow instance information including input, output, error, name of the nodes
func GetDetailedWorkflowInstanceResp(inst model.WorkflowInstanceWithNodes, workflow model.WorkflowWithNodes) (resp DetailedWorkflowInstanceResp, err error) {
	nodesMap := workflow.Nodes.MapByID()
	resp = DetailedWorkflowInstanceResp{
		ID:          inst.ID,
		WorkflowID:  inst.WorkflowID,
		Status:      inst.Status,
		StartNodeID: inst.StartNodeID,
		FailNodeID:  inst.FailNodeID,
		StartTime:   inst.StartTime,
		DurationMs:  inst.DurationMs,
	}

	var sortedNodeIds []string
	iter := validate.NewWorkflowNodeIter(workflow.StartNodeID, workflow.Nodes)
	err = iter.Loop(func(node model.Node) (end bool) {
		// skip foreach node.
		if node.Class == validate.ForeachClass {
			return
		}
		sortedNodeIds = append(sortedNodeIds, node.ID)
		return
	})
	if err != nil {
		return
	}

	data := map[string]WorkflowInstanceNodeData{}
	for _, nodeIns := range inst.Nodes {
		node, ok := nodesMap[nodeIns.NodeID]
		if !ok {
			continue
		}
		if node.Class == validate.ForeachClass {
			return
		}

		var input, output any
		_ = json.Unmarshal(nodeIns.Input, &input)
		_ = json.Unmarshal(nodeIns.Output, &output)
		var nodeErrStr string
		if nodeIns.NodeID == inst.FailNodeID {
			nodeErrStr = inst.Error
		}
		data[nodeIns.NodeID] = WorkflowInstanceNodeData{
			NodeID:   nodeIns.NodeID,
			Input:    input,
			Output:   output,
			NodeName: nodesMap[nodeIns.NodeID].Name,
			Error:    nodeErrStr,
		}
	}

	for _, nodeID := range sortedNodeIds {
		if _, ok := data[nodeID]; !ok {
			continue
		}
		resp.Data = append(resp.Data, data[nodeID])
	}
	return
}

type GetConfirmResp struct {
	Expired         bool                    `json:"expired"`
	WorkflowEnabled bool                    `json:"workflowEnabled"`
	Confirm         model.Confirm           `json:"confirm"`
	Workflow        model.WorkflowWithNodes `json:"workflow"`
}

type GetNodeTestPageDataResp struct {
	InputFields map[string]any `json:"inputFields"`
}

type GetUserToursResp struct {
	Tours map[string]model.UserTour `json:"tours"`
}

func GetUserToursInMap(tours []model.UserTour) GetUserToursResp {
	data := map[string]model.UserTour{}
	for _, tour := range tours {
		data[tour.Path] = tour
	}
	return GetUserToursResp{
		Tours: data,
	}
}

type TestFeatureFlagResp struct {
	Enabled bool `json:"enabled"`
}
