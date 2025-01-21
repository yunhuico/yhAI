package gitlab

import (
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type CreateGroup struct{}

type CreateGroupLabel struct {
	BaseGitlabNode `json:"-"`
	GroupID        int    `json:"groupId"`
	Name           string `json:"name"`
	Color          string `json:"color"`
}

func (c *CreateGroupLabel) Run(ctx *workflow.NodeContext) (any, error) {
	options := &gitlab.CreateGroupLabelOptions{
		Name:  &c.Name,
		Color: &c.Color,
	}
	label, resp, err := c.GetClient().GroupLabels.CreateGroupLabel(c.GroupID, options, gitlab.WithContext(ctx.Context()))
	if err != nil {
		return nil, fmt.Errorf("create group lable error: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return label, nil
}

func (c *CreateGroupLabel) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#createGroupLabel")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CreateGroupLabel)
		},
		InputForm: spec.InputSchema,
	}
}

type ListGroup struct {
	BaseGitlabNode `json:"-"`

	workflow.ListPagination
}

func (l *ListGroup) Run(c *workflow.NodeContext) (any, error) {
	return l.run(c)
}

func (l *ListGroup) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	groups, err := l.run(c)
	if err != nil {
		return
	}

	for _, group := range groups {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: group.Name,
			Value: group.ID,
		})
	}
	return
}

func (l *ListGroup) run(c *workflow.NodeContext) ([]*gitlab.Group, error) {
	opt := gitlab.ListGroupsOptions{
		Owned: &boolTrue,
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
	groups, resp, err := l.GetClient().Groups.ListGroups(&opt, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list group error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return groups, nil
}

func (l *ListGroup) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listGroup")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListGroup)
		},
		InputForm: spec.InputSchema,
	}
}

type ListGroupMembers struct {
	BaseGitlabNode `json:"-"`

	GroupID int    `json:"groupId"`
	Query   string `json:"search"`
}

type ListGroupMembersOutput struct {
	Members []*gitlab.GroupMember `json:"members"`
	Count   int                   `json:"count"`
}

func (l ListGroupMembers) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listGroupMembers")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListGroupMembers)
		},
		InputForm: spec.InputSchema,
	}
}

func (l ListGroupMembers) Run(c *workflow.NodeContext) (output any, err error) {
	opts := &gitlab.ListGroupMembersOptions{}
	if l.Query != "" {
		opts.Query = &l.Query
	}
	members, _, err := l.GetClient().Groups.ListGroupMembers(l.GroupID, opts, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("list group members: %w", err)
		return
	}

	output = ListGroupMembersOutput{
		Members: members,
		Count:   len(members),
	}

	return
}
