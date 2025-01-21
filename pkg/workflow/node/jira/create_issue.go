package jira

import (
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/jira"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func init() {
	workflow.RegistryNodeMeta(&CreateIssue{})
}

type IssueType string

const (
	// IssueTypeEpic 长篇故事
	IssueTypeEpic = "Epic"
	// IssueTypeBug 故障
	IssueTypeBug = "Bug"
	// IssueTypeStory 故事
	IssueTypeStory = "Story"
	// IssueTypeTask 任务
	IssueTypeTask = "Task"
	// IssueTypeSubtask 子任务
	IssueTypeSubtask = "Subtask"
)

type issueInputFields struct {
	baseJiraNode

	// Required fields

	IssueType IssueType `json:"issueType,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	ProjectID string    `json:"projectId,omitempty"`

	// Required for IssueTypeSubtask
	ParentIssueKey string `json:"parentIssueKey,omitempty"`

	// Optional below

	TransitionID string `json:"transitionId,omitempty"` // Only used in issue update
	Description  string `json:"description,omitempty"`
	// We only support the basic case, which only `Impediment` option is chosen
	FlaggedImpediment bool     `json:"FlaggedImpediment,omitempty"`
	ReporterUserID    string   `json:"reporterUserId,omitempty"`
	Labels            []string `json:"labels,omitempty"`
	AssigneeUserID    string   `json:"assigneeUserId,omitempty"`

	// Only IssueTypeEpic supports the following fields

	// e.g. 2019-05-11
	StartDate string `json:"startDate,omitempty"`
	// e.g. 2019-05-11
	DueDate string `json:"dueDate,omitempty"`
	// Valid value:
	// purple, blue, green, teal, yellow, orange, grey,
	// dark_purple, dark_blue, dark_green, dark_teal,
	// dark_yellow, dark_orange, dark_grey
	IssueColor string `json:"issueColor,omitempty"`
}

type CreateIssue struct {
	issueInputFields
}

func (i *CreateIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("createIssue"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CreateIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (i *CreateIssue) Run(c *workflow.NodeContext) (output any, err error) {
	var (
		ctx      = c.Context()
		newIssue jira.CreateIssueReq
	)

	if i.IssueType == "" {
		err = errors.New("issueType is required")
		return
	}
	if i.Summary == "" {
		err = errors.New("summary is required")
		return
	}
	if i.ProjectID == "" {
		err = errors.New("projectId is required")
		return
	}
	if i.IssueType == IssueTypeSubtask && i.ParentIssueKey == "" {
		err = errors.New("parentIssueKey is required for Subtask")
		return
	}

	issueTypes, err := i.client.GetIssueMetadataByProjectID(ctx, i.ProjectID)
	if err != nil {
		err = fmt.Errorf("querying issue types: %w", err)
		return
	}

	issueType, err := findIssueType(issueTypes, i.IssueType)
	if err != nil {
		err = fmt.Errorf("finding issue type from meta: %w", err)
		return
	}

	newIssue.Fields, err = forgeIssueAPIFields(i.issueInputFields, issueType, true)
	if err != nil {
		err = fmt.Errorf("forgeIssueAPIFields: %w", err)
		return
	}

	resp, err := i.client.CreateIssue(ctx, newIssue)
	if err != nil {
		err = fmt.Errorf("creating Jira issue: %w", err)
		return
	}
	output = resp

	return
}

// forgeIssueAPIFields is shared by CreateIssue and UpdateIssue
// so here we regard every field as optional.
func forgeIssueAPIFields(input issueInputFields, meta jira.IssueType, includeIssueTypeField bool) (m map[string]any, err error) {
	m = make(map[string]any)

	if includeIssueTypeField {
		m["issuetype"] = map[string]string{"id": meta.ID}
	}
	if input.Summary != "" {
		m["summary"] = input.Summary
	}
	if input.ProjectID != "" {
		m["project"] = map[string]string{"id": input.ProjectID}
	}
	if input.Description != "" {
		m["description"] = jira.DocText(input.Description)
	}
	if input.ReporterUserID != "" {
		m["reporter"] = map[string]string{"id": input.ReporterUserID}
	}
	if len(input.Labels) > 0 {
		m["labels"] = input.Labels
	}
	if input.AssigneeUserID != "" {
		m["assignee"] = map[string]string{"id": input.AssigneeUserID}
	}

	// Fails silently for optional, non-common fields

	if input.StartDate != "" {
		key := meta.Fields.FindKeyByFieldName("Start date")
		if key != "" {
			m[key] = input.StartDate
		}
	}
	if input.DueDate != "" && meta.Fields.KeyExists("duedate") {
		m["duedate"] = input.DueDate
	}
	if input.IssueColor != "" {
		key := meta.Fields.FindKeyByFieldName("Issue color")
		if key != "" {
			m[key] = input.IssueColor
		}
	}
	if input.ParentIssueKey != "" && meta.Fields.KeyExists("parent") {
		m["parent"] = map[string]string{"key": input.ParentIssueKey}
	}
	if input.FlaggedImpediment {
		key := meta.Fields.FindKeyByFieldName("Flagged")
		if key != "" {
			m[key] = []map[string]string{
				{
					"value": "Impediment",
				},
			}
		}
	}

	return
}

func findIssueType(issueTypes []jira.IssueType, wantType IssueType) (issueType jira.IssueType, err error) {
	var names [2]string
	switch wantType {
	case IssueTypeEpic, "长篇故事":
		names = [2]string{"长篇故事", IssueTypeEpic}
	case IssueTypeBug, "故障":
		names = [2]string{"故障", IssueTypeBug}
	case IssueTypeStory, "故事":
		names = [2]string{"故事", IssueTypeStory}
	case IssueTypeTask, "任务":
		names = [2]string{"任务", IssueTypeTask}
	case IssueTypeSubtask, "子任务":
		names = [2]string{"子任务", IssueTypeSubtask}
	default:
		err = fmt.Errorf("unexpected wantType: %s", wantType)
		return
	}

	for _, item := range issueTypes {
		for _, name := range names {
			if item.Name == name || item.UntranslatedName == name {
				issueType = item
				return
			}
		}
	}

	err = fmt.Errorf("no item typed %s found", wantType)
	return
}
