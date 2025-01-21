package gitlab

import (
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"

	"github.com/itchyny/timefmt-go"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type ListGroupMilestone struct {
	BaseGitlabNode `json:"-"`

	workflow.ListPagination

	GroupID int `json:"groupId"`
}

func (l *ListGroupMilestone) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listGroupMilestone")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListGroupMilestone)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListGroupMilestone) Run(ctx *workflow.NodeContext) (any, error) {
	return l.run(ctx)
}

func (l *ListGroupMilestone) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	milestones, err := l.run(c)
	if err != nil {
		return
	}

	for _, milestone := range milestones {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: milestone.Title,
			Value: milestone.ID,
		})
	}
	return
}

func (l *ListGroupMilestone) run(ctx *workflow.NodeContext) ([]*gitlab.GroupMilestone, error) {
	options := &gitlab.ListGroupMilestonesOptions{}
	if l.Page > 0 {
		options.Page = l.Page
	}
	if l.PerPage > 0 {
		options.PerPage = l.PerPage
	}
	if l.Search != "" {
		options.Search = &l.Search
	}
	options.IncludeParentMilestones = &boolTrue

	milestones, resp, err := l.GetClient().GroupMilestones.ListGroupMilestones(l.GroupID, options, gitlab.WithContext(ctx.Context()))
	if err != nil {
		return nil, fmt.Errorf("list group milestone error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}

	return milestones, nil
}

type CreateGroupMilestone struct {
	BaseGitlabNode `json:"-"`
	GroupID        int    `json:"groupId"`
	StartTime      string `json:"startTime"`
	DueTime        string `json:"dueTime"`
	Title          string `json:"title"`
	Description    string `json:"description"`
}

func (c *CreateGroupMilestone) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#createGroupMilestone")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CreateGroupMilestone)
		},
		InputForm: spec.InputSchema,
	}
}

func (c *CreateGroupMilestone) Run(ctx *workflow.NodeContext) (any, error) {
	options := &gitlab.CreateGroupMilestoneOptions{
		Title:       &c.Title,
		Description: &c.Description,
	}
	if c.StartTime != "" {
		start, err := timefmt.Parse(c.StartTime, "%Y-%m-%d")
		if err != nil {
			return nil, fmt.Errorf("parse start time error: %w", err)
		}
		startTime := gitlab.ISOTime(start)
		options.StartDate = &startTime
	}
	if c.DueTime != "" {
		du, err := timefmt.Parse(c.DueTime, "%Y-%m-%d")
		if err != nil {
			return nil, fmt.Errorf("parse due time error: %w", err)
		}
		dueTime := gitlab.ISOTime(du)
		options.DueDate = &dueTime
	}

	milestone, response, err := c.GetClient().GroupMilestones.CreateGroupMilestone(c.GroupID, options, gitlab.WithContext(ctx.Context()))
	if err != nil {
		return nil, fmt.Errorf("create group milestone error: %w", err)
	}
	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab server response %s", response.Status)
	}

	return milestone, nil
}

type ListProjectMilestone struct {
	BaseGitlabNode `json:"-"`

	workflow.ListPagination

	ProjectID int `json:"projectId"`
}

func (l *ListProjectMilestone) Run(c *workflow.NodeContext) (any, error) {
	return l.run(c)
}

func (l *ListProjectMilestone) run(c *workflow.NodeContext) ([]*gitlab.Milestone, error) {
	options := &gitlab.ListMilestonesOptions{}
	if l.Page > 0 {
		options.Page = l.Page
	}
	if l.PerPage > 0 {
		options.PerPage = l.PerPage
	}
	if l.Search != "" {
		options.Search = &l.Search
	}
	options.IncludeParentMilestones = &boolTrue
	milestones, resp, err := l.GetClient().Milestones.ListMilestones(l.ProjectID, options, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list milestones error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return milestones, nil
}

func (l *ListProjectMilestone) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	milestones, err := l.run(c)
	if err != nil {
		return
	}

	for _, milestone := range milestones {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: milestone.Title,
			Value: milestone.ID,
		})
	}
	return
}

func (l *ListProjectMilestone) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listProjectMilestone")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListProjectMilestone)
		},
		InputForm: spec.InputSchema,
	}
}
