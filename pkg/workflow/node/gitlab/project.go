package gitlab

import (
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type ListProject struct {
	BaseGitlabNode `json:"-"`

	workflow.ListPagination
}

func (e *ListProject) Run(c *workflow.NodeContext) (any, error) {
	return e.run(c)
}

func (e *ListProject) run(c *workflow.NodeContext) ([]*gitlab.Project, error) {
	opt := &gitlab.ListProjectsOptions{
		Membership: &boolTrue,
	}
	if e.Page > 0 {
		opt.Page = e.Page
	}
	if e.PerPage > 0 {
		opt.PerPage = e.PerPage
	}
	if e.Search != "" {
		opt.Search = &e.Search
	}

	projects, resp, err := e.GetClient().Projects.ListProjects(opt, gitlab.WithContext(c.Context()))
	if err != nil {
		return nil, fmt.Errorf("list project error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab server response %s", resp.Status)
	}
	return projects, nil
}

func (e *ListProject) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	projects, err := e.run(c)
	if err != nil {
		return
	}
	for _, project := range projects {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: project.NameWithNamespace,
			Value: project.ID,
		})
	}
	return
}

func (e *ListProject) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#listProject")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListProject)
		},
		InputForm: spec.InputSchema,
	}
}
