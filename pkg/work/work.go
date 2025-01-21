// Package work implements pending work delivery between
// trigger components and workers.
//
// Design doc: https://jihulab.com/ultrafox/ultrafox/-/issues/184
package work

import "errors"

// Work is a pending work triggered by Triggers,
// related to a workflow.
type Work struct {
	// Unique identity of the work,
	// corresponding with the primary key of workflow_instances
	//
	// Auto-generated if Resume field is false, which is always the case of brand-new work.
	// To resume paused work, the id will be corresponding with the existing record of workflow_instances.
	ID string `json:"id"`
	// corresponding with the primary key of workflows
	WorkflowID string `json:"workflowId"`
	// is the work resuming a paused one?
	Resume bool `json:"resume"`
	// Workflow execution timeout in seconds
	MaxDurationSeconds int `json:"maxDurationSeconds"`
	// How many steps can a workflow executes at most?
	MaxSteps int `json:"maxSteps"`
	// which node to start or resume?
	StartNodeID string `json:"startNodeId"`
	// used in brand-new work, the starting payload
	StartNodePayload []byte `json:"startNodePayload"`
}

func (w Work) validate() (err error) {
	if w.ID == "" {
		err = errors.New("work id is missing")
		return
	}
	if w.WorkflowID == "" {
		err = errors.New("workflowId is missing")
		return
	}
	if w.StartNodeID == "" {
		err = errors.New("startNodeId is missing")
		return
	}
	if !w.Resume && w.StartNodePayload == nil {
		err = errors.New("startNodePayload is missing when resume is false")
		return
	}

	return
}
