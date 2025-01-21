package gitlab

import (
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type ListUser struct {
	BaseGitlabNode `json:"-"`

	workflow.ListPagination
}

func (l *ListUser) Run(c *workflow.NodeContext) (any, error) {
	return l.run(c)
}

func (l *ListUser) run(c *workflow.NodeContext) ([]*gitlab.User, error) {
	opt := gitlab.ListUsersOptions{}
	if l.Page > 0 {
		opt.Page = l.Page
	}
	if l.PerPage > 0 {
		opt.PerPage = l.PerPage
	}
	if l.Search != "" {
		opt.Search = &l.Search
	}
	users, resp, err := l.GetClient().Users.ListUsers(&opt, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list users error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return users, nil
}

func (l *ListUser) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	users, err := l.run(c)
	if err != nil {
		return
	}

	for _, user := range users {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: user.Name,
			Value: user.ID,
		})
	}
	return
}

func (l *ListUser) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listUser")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListUser)
		},
		InputForm: spec.InputSchema,
	}
}
