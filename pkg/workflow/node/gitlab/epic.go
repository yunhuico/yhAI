package gitlab

import (
	"fmt"
	"net/http"

	"github.com/itchyny/timefmt-go"
	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type CreateEpic struct {
	BaseGitlabNode `json:"-"`
	GroupID        int            `json:"groupId"`
	Title          string         `json:"title"`
	Description    *string        `json:"description"`
	Labels         *gitlab.Labels `json:"labels"`
	StartDateFixed string         `json:"startDateFixed"`
	DueDateFixed   string         `json:"dueDateFixed"`
}

func (e *CreateEpic) Run(c *workflow.NodeContext) (any, error) {
	options := &gitlab.CreateEpicOptions{
		Title:       &e.Title,
		Description: e.Description,
		Labels:      e.Labels,
	}
	if e.StartDateFixed != "" {
		startDateTime, err := timefmt.Parse(e.StartDateFixed, "%Y-%m-%d")
		if err != nil {
			return nil, fmt.Errorf("parse start date error: %w", err)
		}
		startDate := gitlab.ISOTime(startDateTime)
		options.StartDateIsFixed = &boolTrue
		options.StartDateFixed = &startDate
	}
	if e.DueDateFixed != "" {
		dueDateTime, err := timefmt.Parse(e.DueDateFixed, "%Y-%m-%d")
		if err != nil {
			return nil, fmt.Errorf("parse due date error: %w", err)
		}
		dueDate := gitlab.ISOTime(dueDateTime)
		options.DueDateIsFixed = &boolTrue
		options.DueDateFixed = &dueDate
	}

	epic, resp, err := e.GetClient().Epics.CreateEpic(e.GroupID, options)
	if err != nil {
		return nil, fmt.Errorf("create epic error: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}

	return epic, nil
}

func (e *CreateEpic) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#createEpic")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CreateEpic)
		},
		InputForm: spec.InputSchema,
	}
}

type UpdateEpic struct {
	BaseGitlabNode `json:"-"`
	GroupID        int            `json:"groupId"`
	EpicID         int            `json:"epicId"`
	Title          string         `json:"title"`
	Description    *string        `json:"description"`
	Labels         *gitlab.Labels `json:"labels"`
	StartDateFixed string         `json:"startDateFixed"`
	DueDateFixed   string         `json:"dueDateFixed"`
}

func (e *UpdateEpic) Run(c *workflow.NodeContext) (any, error) {
	options := &gitlab.UpdateEpicOptions{
		Title: &e.Title,
	}
	if e.Description != nil && *e.Description != "" {
		options.Description = e.Description
	}

	if e.StartDateFixed != "" {
		startDateTime, err := timefmt.Parse(e.StartDateFixed, "%Y-%m-%d")
		if err != nil {
			return nil, fmt.Errorf("parse start date error: %w", err)
		}
		startDate := gitlab.ISOTime(startDateTime)
		options.StartDateIsFixed = &boolTrue
		options.StartDateFixed = &startDate
	}
	if e.DueDateFixed != "" {
		dueDateTime, err := timefmt.Parse(e.DueDateFixed, "%Y-%m-%d")
		if err != nil {
			return nil, fmt.Errorf("parse due date error: %w", err)
		}
		dueDate := gitlab.ISOTime(dueDateTime)
		options.DueDateIsFixed = &boolTrue
		options.DueDateFixed = &dueDate
	}

	epics, resp, err := e.GetClient().Epics.UpdateEpic(e.GroupID, e.EpicID, options)
	if err != nil {
		return nil, fmt.Errorf("update group epic error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return epics, nil
}

func (e *UpdateEpic) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#updateEpic")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(UpdateEpic)
		},
		InputForm: spec.InputSchema,
	}
}

type ListGroupEpic struct {
	BaseGitlabNode `json:"-"`
	workflow.ListPagination
	GroupID int    `json:"groupId"`
	Search  string `json:"search"`
	State   string `json:"state"`
}

func (e *ListGroupEpic) Run(c *workflow.NodeContext) (any, error) {
	return e.run(c)
}

func (e *ListGroupEpic) run(c *workflow.NodeContext) ([]*gitlab.Epic, error) {
	options := &gitlab.ListGroupEpicsOptions{}
	if e.Search != "" {
		options.Search = &e.Search
	}
	if e.State != "" {
		options.State = &e.State
	}
	if e.Page > 0 {
		options.Page = e.Page
	}
	if e.PerPage > 0 {
		options.PerPage = e.PerPage
	}
	epics, resp, err := e.GetClient().Epics.ListGroupEpics(e.GroupID, options, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list group epics error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return epics, nil
}

func (e *ListGroupEpic) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	epics, err := e.run(c)
	if err != nil {
		return
	}

	for _, epic := range epics {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: epic.Title,
			Value: epic.IID,
		})
	}
	return
}

func (e *ListGroupEpic) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listGroupEpic")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListGroupEpic)
		},
		InputForm: spec.InputSchema,
	}
}
