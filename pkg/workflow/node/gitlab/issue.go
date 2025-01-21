package gitlab

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type AddProjectIssueLabel struct {
	BaseGitlabNode `json:"-"`

	ProjectID int           `json:"projectId"`
	IssueID   int           `json:"issueId"`
	AddLabels gitlab.Labels `json:"addLabels"`
}

func (g *AddProjectIssueLabel) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#addProjectIssueLabel")

	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(AddProjectIssueLabel)
		},
		InputForm: spec.InputSchema,
	}
}

type addProjectIssueLabelOutput struct {
	CurrentLabels gitlab.Labels `json:"currentLabels"`
}

func (g *AddProjectIssueLabel) Run(c *workflow.NodeContext) (any, error) {
	c.Debug("start run", log.Any("raw", g))

	options := &gitlab.UpdateIssueOptions{
		AddLabels: &g.AddLabels,
	}
	issue, response, err := g.GetClient().Issues.UpdateIssue(g.ProjectID, g.IssueID, options, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("create issue failed: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", response.Status)
	}

	return addProjectIssueLabelOutput{issue.Labels}, nil
}

type IssueLink struct {
	BaseGitlabNode  `json:"-"`
	ProjectID       int `json:"projectId"`
	IssueID         int `json:"issueId"`
	TargetProjectId int `json:"targetProjectId"`
	TargetIssueIId  int `json:"targetIssueIId"`
}

func (s *IssueLink) Run(c *workflow.NodeContext) (any, error) {
	targetProjectID := strconv.Itoa(s.TargetProjectId)
	targetIssueIID := strconv.Itoa(s.TargetIssueIId)
	options := &gitlab.CreateIssueLinkOptions{
		TargetProjectID: &targetProjectID,
		TargetIssueIID:  &targetIssueIID,
	}
	_, resp, err := s.GetClient().IssueLinks.CreateIssueLink(s.ProjectID, s.IssueID, options, gitlab.WithContext(c.Context()))
	if err != nil {
		// if statusCode is 406, indicate message: Issue(s) already assigned.
		if resp.StatusCode == http.StatusConflict {
			return map[string]any{"success": 1}, nil
		}
		return nil, fmt.Errorf("create issue link error: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return map[string]any{"success": 1}, nil
}

func (s *IssueLink) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#issueLink")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(IssueLink)
		},
		InputForm: spec.InputSchema,
	}
}

type ListGroupIssue struct {
	BaseGitlabNode `json:"-"`
	GroupID        int            `json:"groupId"`
	ProjectID      int            `json:"projectId"`
	Labels         *gitlab.Labels `json:"labels"`
	State          string         `json:"state"`
	AuthorID       int            `json:"authorId"`
	AssigneeID     int            `json:"assigneeId"`
}

func (s *ListGroupIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listGroupIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListGroupIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *ListGroupIssue) Run(c *workflow.NodeContext) (any, error) {
	options := &gitlab.ListGroupIssuesOptions{
		Labels: s.Labels,
	}
	if s.State != "" {
		options.State = &s.State
	}
	if s.Labels != nil && len(*s.Labels) > 0 {
		options.Labels = s.Labels
	}
	if s.AuthorID > 0 {
		options.AuthorID = &s.AuthorID
	}
	if s.AssigneeID > 0 {
		options.AssigneeID = gitlab.AssigneeID(options.AssigneeID)
	}
	issues, response, err := s.GetClient().Issues.ListGroupIssues(s.GroupID, options, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list group issue failed: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", response.Status)
	}

	return issues, nil
}

type CommentProjectIssue struct {
	BaseGitlabNode `json:"-"`

	IssueID   int    `json:"issueId"`
	ProjectID int    `json:"projectId"`
	Body      string `json:"body"`
}

func (s *CommentProjectIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#commentProjectIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CommentProjectIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *CommentProjectIssue) Run(c *workflow.NodeContext) (any, error) {
	options := &gitlab.CreateIssueNoteOptions{
		Body: &s.Body,
	}
	note, response, err := s.GetClient().Notes.CreateIssueNote(s.ProjectID, s.IssueID, options, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list group issue failed: %w", err)
	}
	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab server response %s", response.Status)
	}

	return note, nil
}

type CloseProjectIssue struct {
	BaseGitlabNode `json:"-"`

	IssueID   int `json:"issueId"`
	ProjectID int `json:"projectId"`
}

func (s *CloseProjectIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#closeProjectIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CloseProjectIssue)
		},
		InputForm: spec.InputSchema,
	}
}

var closeEvent = "close"

func (s *CloseProjectIssue) Run(c *workflow.NodeContext) (any, error) {
	note, response, err := s.GetClient().Issues.UpdateIssue(s.ProjectID, s.IssueID, &gitlab.UpdateIssueOptions{
		StateEvent: &closeEvent,
	})
	if err != nil {
		return nil, fmt.Errorf("close issue failed: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", response.Status)
	}

	return note, nil
}

type CreateProjectIssue struct {
	BaseGitlabNode `json:"-"`

	ProjectID   int            `json:"projectId"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Labels      *gitlab.Labels `json:"labels"`
	MilestoneID int            `json:"milestoneId"`
	AssigneeIDs []int          `json:"assigneeIds"`
}

func (s *CreateProjectIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#createProjectIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CreateProjectIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *CreateProjectIssue) Run(c *workflow.NodeContext) (any, error) {
	options := &gitlab.CreateIssueOptions{
		Title:       &s.Title,
		Description: &s.Description,
		AssigneeIDs: &s.AssigneeIDs,
		MilestoneID: &s.MilestoneID,
		Labels:      s.Labels,
	}
	issue, resp, err := s.GetClient().Issues.CreateIssue(s.ProjectID, options, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("create issue error: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return issue, nil
}

type UpdateProjectIssue struct {
	BaseGitlabNode `json:"-"`
	ProjectID      int           `json:"projectId"`
	IssueID        int           `json:"issueId"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	AddLabels      gitlab.Labels `json:"addLabels"`
	MilestoneID    int           `json:"milestoneId"`
	AssigneeIDs    []int         `json:"assigneeIds"`
}

func (s *UpdateProjectIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#updateProjectIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(UpdateProjectIssue)
		},
		InputForm: spec.InputSchema,
	}
}

func (s *UpdateProjectIssue) Run(c *workflow.NodeContext) (any, error) {
	options := &gitlab.UpdateIssueOptions{}
	if s.Title != "" {
		options.Title = &s.Title
	}
	if s.Description != "" {
		options.Description = &s.Description
	}
	if len(s.AddLabels) > 0 {
		options.AddLabels = &s.AddLabels
	}
	if s.MilestoneID > 0 {
		options.MilestoneID = &s.MilestoneID
	}
	if len(s.AssigneeIDs) > 0 {
		options.AssigneeIDs = &s.AssigneeIDs
	}

	issue, resp, err := s.GetClient().Issues.UpdateIssue(s.ProjectID, s.IssueID, options)
	if err != nil {
		return nil, fmt.Errorf("create issue error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return issue, nil
}

type ListIssue struct {
	BaseGitlabNode `json:"-"`

	ProjectID  int           `json:"projectId"`
	Labels     gitlab.Labels `json:"labels"`
	AuthorID   int           `json:"authorId"`
	AssigneeID int           `json:"assigneeId"`
	State      string        `json:"state"`

	workflow.ListPagination
}

func (l *ListIssue) Run(c *workflow.NodeContext) (any, error) {
	return l.run(c)
}

func (l *ListIssue) run(c *workflow.NodeContext) ([]*gitlab.Issue, error) {
	opt := gitlab.ListProjectIssuesOptions{}
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
	issues, resp, err := l.GetClient().Issues.ListProjectIssues(l.ProjectID, &opt, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list issue error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return issues, nil
}

func (l *ListIssue) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	issues, err := l.run(c)
	if err != nil {
		return
	}

	for _, issue := range issues {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: issue.Title,
			Value: issue.IID,
		})
	}
	return
}

func (l *ListIssue) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listIssue")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListIssue)
		},
		InputForm: spec.InputSchema,
	}
}
