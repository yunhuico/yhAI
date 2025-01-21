package gitlab

import (
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type GetProjectMergeRequest struct {
	BaseGitlabNode `json:"-"`

	ProjectID      int `json:"projectId"`
	MergeRequestID int `json:"mergeRequestId"`
}

func (g GetProjectMergeRequest) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#getProjectMergeRequest")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(GetProjectMergeRequest)
		},
		InputForm: spec.InputSchema,
	}
}

func (g GetProjectMergeRequest) Run(c *workflow.NodeContext) (mergeRequest any, err error) {
	mergeRequest, _, err = g.client.MergeRequests.GetMergeRequest(g.ProjectID, g.MergeRequestID, nil, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get merge request failed: %w", err)
		return
	}
	return
}

var _ workflow.QueryFieldResultProvider = (*ListProjectMergeRequests)(nil)

type ListProjectMergeRequests struct {
	BaseGitlabNode `json:"-"`

	ProjectID  int           `json:"projectId"`
	Labels     gitlab.Labels `json:"labels"`
	AuthorID   int           `json:"authorId"`
	AssigneeID int           `json:"assigneeId"`
	State      string        `json:"state"`

	workflow.ListPagination
}

func (l ListProjectMergeRequests) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listProjectMergeRequests")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListProjectMergeRequests)
		},
		InputForm: spec.InputSchema,
	}
}

func (l ListProjectMergeRequests) Run(c *workflow.NodeContext) (any, error) {
	return l.run(c)
}

func (l ListProjectMergeRequests) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	mergeRequests, err := l.run(c)
	if err != nil {
		err = fmt.Errorf("query field result list: %w", err)
		return
	}
	result = workflow.QueryFieldResult{
		Items: make([]workflow.QueryFieldItem, len(mergeRequests)),
	}
	for i, mergeRequest := range mergeRequests {
		result.Items[i] = workflow.QueryFieldItem{
			Label: fmt.Sprintf("[%s] %s", mergeRequest.State, mergeRequest.Title),
			Value: mergeRequest.IID,
		}
	}
	return
}

func (l ListProjectMergeRequests) run(c *workflow.NodeContext) (mergeRequests []*gitlab.MergeRequest, err error) {
	opt := &gitlab.ListProjectMergeRequestsOptions{
		Search: &l.Search,
	}
	if len(l.Labels) > 0 {
		opt.Labels = &l.Labels
	}
	if l.AuthorID > 0 {
		opt.AuthorID = &l.AuthorID
	}
	if l.AssigneeID > 0 {
		opt.AssigneeID = gitlab.AssigneeID(l.AssigneeID)
	}
	if l.State != "" {
		opt.State = &l.State
	}
	opt.Page = l.Page
	opt.PerPage = l.PerPage
	mergeRequests, _, err = l.client.MergeRequests.ListProjectMergeRequests(l.ProjectID, opt, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("list project merge requests: %w", err)
		return
	}
	return
}

type UpdateProjectMergeRequest struct {
	BaseGitlabNode `json:"-"`

	ProjectID      int           `json:"projectId"`
	MergeRequestID int           `json:"mergeRequestId"`
	Description    string        `json:"description"`
	AddLabels      gitlab.Labels `json:"addLabels"`
	MilestoneID    int           `json:"milestoneId"`
	AssigneeIDs    []int         `json:"assigneeIds"`
}

func (u UpdateProjectMergeRequest) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#updateProjectMergeRequest")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(UpdateProjectMergeRequest)
		},
		InputForm: spec.InputSchema,
	}
}

func (u UpdateProjectMergeRequest) Run(c *workflow.NodeContext) (any, error) {
	opt := &gitlab.UpdateMergeRequestOptions{}
	if u.Description != "" {
		opt.Description = &u.Description
	}
	if u.MilestoneID != 0 {
		opt.MilestoneID = &u.MilestoneID
	}
	if len(u.AssigneeIDs) > 0 {
		opt.AssigneeIDs = &u.AssigneeIDs
	}
	if len(u.AddLabels) > 0 {
		opt.AddLabels = &u.AddLabels
	}

	_, resp, err := u.client.MergeRequests.UpdateMergeRequest(u.ProjectID, u.MergeRequestID, opt)
	if err != nil {
		return nil, fmt.Errorf("update merge request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return map[string]any{"success": true}, nil
}

type CommentProjectMergeRequest struct {
	BaseGitlabNode `json:"-"`

	ProjectID      int    `json:"projectId"`
	MergeRequestID int    `json:"mergeRequestId"`
	Body           string `json:"body"`
}

func (r CommentProjectMergeRequest) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#commentProjectMergeRequest")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CommentProjectMergeRequest)
		},
		InputForm: spec.InputSchema,
	}
}

func (r CommentProjectMergeRequest) Run(c *workflow.NodeContext) (any, error) {
	_, _, err := r.client.Notes.CreateMergeRequestNote(r.ProjectID, r.MergeRequestID, &gitlab.CreateMergeRequestNoteOptions{
		Body: &r.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("creating merge request comment: %w", err)
	}
	return map[string]any{"success": true}, nil
}

type GetProjectIssue struct {
	BaseGitlabNode `json:"-"`

	ProjectID int `json:"projectId"`
	IssueID   int `json:"issueId"`
}

func (g GetProjectIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#getProjectIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(GetProjectIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (g GetProjectIssue) Run(c *workflow.NodeContext) (issue any, err error) {
	issue, _, err = g.client.Issues.GetIssue(g.ProjectID, g.IssueID, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get project issue: %w", err)
		return
	}
	return
}

type ListGroupMergeRequests struct {
	BaseGitlabNode `json:"-"`

	GroupID    int           `json:"groupId"`
	Labels     gitlab.Labels `json:"labels"`
	AuthorID   int           `json:"authorId"`
	AssigneeID int           `json:"assigneeId"`
	State      string        `json:"state"`

	workflow.ListPagination
}

func (l ListGroupMergeRequests) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listGroupMergeRequests")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListGroupMergeRequests)
		},
		InputForm: spec.InputSchema,
	}
}

func (l ListGroupMergeRequests) Run(c *workflow.NodeContext) (any, error) {
	return l.run(c)
}

func (l ListGroupMergeRequests) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	mergeRequests, err := l.run(c)
	if err != nil {
		err = fmt.Errorf("query field result list: %w", err)
		return
	}
	result = workflow.QueryFieldResult{
		Items: make([]workflow.QueryFieldItem, len(mergeRequests)),
	}
	for i, mergeRequest := range mergeRequests {
		result.Items[i] = workflow.QueryFieldItem{
			Label: fmt.Sprintf("[%s] %s", mergeRequest.State, mergeRequest.Title),
			Value: mergeRequest.IID,
		}
	}
	return
}

func (l ListGroupMergeRequests) run(c *workflow.NodeContext) (mergeRequests []*gitlab.MergeRequest, err error) {
	opt := &gitlab.ListGroupMergeRequestsOptions{
		Search: &l.Search,
	}
	if l.Page > 0 {
		opt.Page = l.Page
	}
	if l.PerPage > 0 {
		opt.PerPage = l.PerPage
	}
	if l.Search != "" {
		opt.Search = &l.Search
	}
	if len(l.Labels) > 0 {
		opt.Labels = &l.Labels
	}
	if l.AuthorID > 0 {
		opt.AuthorID = &l.AuthorID
	}
	if l.AssigneeID > 0 {
		opt.AssigneeID = gitlab.AssigneeID(l.AssigneeID)
	}
	if l.State != "" {
		opt.State = &l.State
	}
	mergeRequests, _, err = l.client.MergeRequests.ListGroupMergeRequests(l.GroupID, opt, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("list project merge requests: %w", err)
		return
	}
	return
}

type ListMergeRequestsRelatedToIssue struct {
	BaseGitlabNode `json:"-"`

	ProjectID int `json:"projectId"`
	IssueID   int `json:"issueId"`
}

type ListIssueRelatedMergeRequestsOutput struct {
	Count         int                    `json:"count"`
	MergeRequests []*gitlab.MergeRequest `json:"mergeRequests"`
}

func (l ListMergeRequestsRelatedToIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listMergeRequestsRelatedToIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListMergeRequestsRelatedToIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (l ListMergeRequestsRelatedToIssue) Run(c *workflow.NodeContext) (output any, err error) {
	mergeRequests, _, err := l.GetClient().Issues.ListMergeRequestsRelatedToIssue(l.ProjectID, l.IssueID, nil, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("list issue related merge requests: %w", err)
		return
	}
	output = ListIssueRelatedMergeRequestsOutput{
		Count:         len(mergeRequests),
		MergeRequests: mergeRequests,
	}
	return
}
