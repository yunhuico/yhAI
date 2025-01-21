package jira

import (
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/jira"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func init() {
	workflow.RegistryNodeMeta(&UpdateIssue{})
}

type UpdateIssue struct {
	IssueKeyOrID string `json:"issueKeyOrId"`

	issueInputFields
}

func (i *UpdateIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("updateIssue"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(UpdateIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (i *UpdateIssue) Run(c *workflow.NodeContext) (output any, err error) {
	var (
		ctx           = c.Context()
		updatePayload jira.UpdateIssueReq
	)

	issueTypes, err := i.client.GetIssueMetadataByProjectID(ctx, i.ProjectID)
	if err != nil {
		err = fmt.Errorf("querying issue types: %w", err)
		return
	}

	originIssue, err := i.client.GetIssue(ctx, i.IssueKeyOrID)
	if err != nil {
		err = fmt.Errorf("querying original issue %q from Jira: %w", i.IssueKeyOrID, err)
		return
	}

	issueType, err := findIssueType(issueTypes, IssueType(originIssue.Fields.IssueType.Name))
	if err != nil {
		err = fmt.Errorf("finding issue type %q from meta: %w", originIssue.Fields.IssueType.Name, err)
		return
	}

	updatePayload.Fields, err = forgeIssueAPIFields(i.issueInputFields, issueType, false)
	if err != nil {
		err = fmt.Errorf("forgeIssueAPIFields: %w", err)
		return
	}

	if len(updatePayload.Fields) > 0 {
		err = i.client.UpdateIssue(ctx, i.IssueKeyOrID, updatePayload)
		if err != nil {
			err = fmt.Errorf("updating Jira issue %q: %w", i.IssueKeyOrID, err)
			return
		}
	}

	if i.TransitionID != "" {
		err = i.client.TransitionIssue(ctx, i.IssueKeyOrID, i.TransitionID)
		if err != nil {
			err = fmt.Errorf("transitioning issue %q: %w", i.IssueKeyOrID, err)
			return
		}
	}

	output = updateIssueResp{
		ID:   originIssue.ID,
		Key:  originIssue.Key,
		Self: originIssue.Self,
	}

	return
}

type updateIssueResp struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}
