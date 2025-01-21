package payload

import (
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

// EditWorkflowReq for workflow create api. when init a workflow, user just submit a name.
type EditWorkflowReq struct {
	Name        string `json:"name" validate:"required,max=100,min=1"`
	Description string `json:"description" validate:"max=300"`
}

type RunWorkflowReq struct {
	// NodeID the start node of workflow you want run.
	NodeID string `json:"nodeId"`
	// UseExternalInput if pass the initial input by frontend, should make it's true.
	UseExternalInput bool `json:"useExternalInput"`
	// The external input raw data.
	Input []byte `json:"input"`
}

type SearchLogReq struct {
	WorkflowName string                       `form:"workflowName"`
	StartTime    time.Time                    `form:"startTime" time_format:"2006-01-02 15:04:05" time_utc:"8"`
	EndTime      time.Time                    `form:"endTime"  time_format:"2006-01-02 15:04:05" time_utc:"8"`
	Status       model.WorkflowInstanceStatus `form:"status"`
	IsAsc        bool                         `form:"isAsc"`
	IconNumber   int                          `form:"iconNumber,default=5"`
}
