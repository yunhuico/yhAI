package jira

import (
	"errors"
	"fmt"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/jira"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func init() {
	workflow.RegistryNodeMeta(&ListProjectKeys{})
	workflow.RegistryNodeMeta(&ListProjectIDs{})
	workflow.RegistryNodeMeta(&ListAssignableUsers{})
	workflow.RegistryNodeMeta(&SearchProjectIssues{})
	workflow.RegistryNodeMeta(&ListIssueTransitions{})
}

type ListProjectKeys struct {
	baseJiraNode

	workflow.ListPagination
}

func (l *ListProjectKeys) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("listProjectKeys"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListProjectKeys)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListProjectKeys) Run(c *workflow.NodeContext) (output any, err error) {
	var perPage = l.PerPage
	if perPage == 0 {
		perPage = 20 // default is 20 in frontend
	}

	var resp jira.ProjectSearchResp
	resp, err = l.run(c, perPage)
	if err != nil {
		err = fmt.Errorf("running query listProjectKeys: %w", err)
		return
	}

	output = resp.Values
	return
}

func (l *ListProjectKeys) run(c *workflow.NodeContext, perPage int) (output jira.ProjectSearchResp, err error) {
	resp, err := l.client.ProjectSearch(c.Context(), jira.ProjectSearchOpt{
		StartAt:    max(l.Page-1, 0) * perPage,
		MaxResults: perPage,
		Query:      l.Search,
	})
	if err != nil {
		err = fmt.Errorf("searching Jira projects: %w", err)
		return
	}

	output = resp
	return
}

func (l *ListProjectKeys) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	var perPage = l.PerPage
	if perPage == 0 {
		perPage = 20 // default is 20 in frontend
	}

	var resp jira.ProjectSearchResp
	resp, err = l.run(c, perPage)
	if err != nil {
		err = fmt.Errorf("running query listProjectKeys: %w", err)
		return
	}

	items := make([]workflow.QueryFieldItem, 0, len(resp.Values))
	for _, item := range resp.Values {
		items = append(items, workflow.QueryFieldItem{
			Label: item.Name,
			Value: item.Key,
		})
	}

	result = workflow.QueryFieldResult{
		Items: items,
	}
	if max(l.Page, 1)*perPage > resp.Total {
		result.NoMore = true
	}

	return
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

type ListProjectIDs struct {
	baseJiraNode

	workflow.ListPagination
}

func (l *ListProjectIDs) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("listProjectIDs"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListProjectIDs)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListProjectIDs) Run(c *workflow.NodeContext) (output any, err error) {
	var perPage = l.PerPage
	if perPage == 0 {
		perPage = 20 // default is 20 in frontend
	}

	var resp jira.ProjectSearchResp
	resp, err = l.run(c, perPage)
	if err != nil {
		err = fmt.Errorf("running query listProjectKeys: %w", err)
		return
	}

	output = resp.Values
	return
}

func (l *ListProjectIDs) run(c *workflow.NodeContext, perPage int) (output jira.ProjectSearchResp, err error) {
	resp, err := l.client.ProjectSearch(c.Context(), jira.ProjectSearchOpt{
		StartAt:    max(l.Page-1, 0) * perPage,
		MaxResults: perPage,
		Query:      l.Search,
	})
	if err != nil {
		err = fmt.Errorf("searching Jira projects: %w", err)
		return
	}

	output = resp
	return
}

func (l *ListProjectIDs) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	var perPage = l.PerPage
	if perPage == 0 {
		perPage = 20 // default is 20 in frontend
	}

	var resp jira.ProjectSearchResp
	resp, err = l.run(c, perPage)
	if err != nil {
		err = fmt.Errorf("running query listProjectKeys: %w", err)
		return
	}

	items := make([]workflow.QueryFieldItem, 0, len(resp.Values))
	for _, item := range resp.Values {
		items = append(items, workflow.QueryFieldItem{
			Label: item.Name,
			Value: item.ID,
		})
	}

	result = workflow.QueryFieldResult{
		Items: items,
	}
	if max(l.Page, 1)*perPage > resp.Total {
		result.NoMore = true
	}

	return
}

type ListAssignableUsers struct {
	baseJiraNode

	workflow.ListPagination
	ProjectID string `json:"projectId"`
}

func (l *ListAssignableUsers) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	if l.PerPage == 0 {
		l.PerPage = 20
	}

	if l.ProjectID == "" {
		err = errors.New("project id is required")
		return
	}

	users, err := l.client.FindAssignableUsers(c.Context(), jira.FindAssignableUsersParam{
		ProjectKeyOrID: l.ProjectID,
		StartAt:        max(l.Page-1, 0) * l.PerPage,
		MaxResults:     l.PerPage,
		Query:          l.Search,
	})
	if err != nil {
		err = fmt.Errorf("searching Jira assignable users: %w", err)
		return
	}

	items := make([]workflow.QueryFieldItem, 0, len(users))
	for _, user := range users {
		items = append(items, workflow.QueryFieldItem{
			Label: user.DisplayName,
			Value: user.AccountID,
		})
	}

	result = workflow.QueryFieldResult{
		Items:  items,
		NoMore: len(items) == 0,
	}

	return
}

func (l *ListAssignableUsers) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("listAssignableUsers"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListAssignableUsers)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListAssignableUsers) Run(c *workflow.NodeContext) (output any, err error) {
	err = errors.New("not reached")
	return
}

type SearchProjectIssues struct {
	baseJiraNode

	workflow.ListPagination
	ProjectID string `json:"projectId"`
}

func (l *SearchProjectIssues) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	if l.PerPage == 0 {
		l.PerPage = 20
	}

	if l.ProjectID == "" {
		err = errors.New("project id is required")
		return
	}

	l.Search = strings.TrimSpace(l.Search)
	l.Search = strings.ReplaceAll(l.Search, `"`, "")

	var jql string
	if l.Search == "" {
		jql = fmt.Sprintf(`project = "%s" order by created desc`, l.ProjectID)
	} else {
		jql = fmt.Sprintf(`project = "%s" and text ~ "%s" order by created desc`, l.ProjectID, l.Search)
	}

	matches, err := l.client.IssueSearch(c.Context(), jira.IssueSearchOpt{
		JQL:        jql,
		StartAt:    max(l.Page-1, 0) * l.PerPage,
		MaxResults: l.PerPage,
	})
	if err != nil {
		err = fmt.Errorf("searching issues: %w", err)
		return
	}

	items := make([]workflow.QueryFieldItem, 0, len(matches.Issues))
	for _, item := range matches.Issues {
		items = append(items, workflow.QueryFieldItem{
			Label: item.Fields.Summary,
			Value: item.Key,
		})
	}

	result = workflow.QueryFieldResult{
		Items:  items,
		NoMore: len(items) == 0,
	}

	return
}

func (l *SearchProjectIssues) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("searchProjectIssues"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(SearchProjectIssues)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *SearchProjectIssues) Run(c *workflow.NodeContext) (output any, err error) {
	err = errors.New("not reached")
	return
}

type ListIssueTransitions struct {
	baseJiraNode

	workflow.ListPagination
	IssueKeyOrID string `json:"issueKeyOrId"`
}

func (l *ListIssueTransitions) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	if l.IssueKeyOrID == "" {
		err = errors.New("IssueKeyOrID is required")
		return
	}

	transitions, err := l.client.GetValidTransitionOfIssue(c.Context(), l.IssueKeyOrID)
	if err != nil {
		err = fmt.Errorf("getting valid transition of issue: %w", err)
		return
	}

	items := make([]workflow.QueryFieldItem, 0, len(transitions))
	for _, status := range transitions {
		items = append(items, workflow.QueryFieldItem{
			Label: status.Name,
			Value: status.ID,
		})
	}

	result = workflow.QueryFieldResult{
		Items:  items,
		NoMore: len(items) == 0,
	}

	return
}

func (l *ListIssueTransitions) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("listIssueTransitions"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListIssueTransitions)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListIssueTransitions) Run(c *workflow.NodeContext) (output any, err error) {
	err = errors.New("not reached")
	return
}
