package gitlab

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type UpdateRelease struct {
	BaseGitlabNode `json:"-"`

	ProjectID   int      `json:"projectId"`
	TagName     string   `json:"tagName"`
	Description string   `json:"description"`
	Milestones  []string `json:"milestones"`
}

type UpdateReleaseOutput struct {
	Success bool `json:"success"`
}

func (u UpdateRelease) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#updateRelease")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(UpdateRelease)
		},
		InputForm: spec.InputSchema,
	}
}

func (u UpdateRelease) Run(c *workflow.NodeContext) (output any, err error) {
	opts := &gitlab.UpdateReleaseOptions{}
	if u.Description != "" {
		opts.Description = &u.Description
	}
	if len(u.Milestones) > 0 {
		opts.Milestones = &u.Milestones
	}
	_, _, err = u.GetClient().Releases.UpdateRelease(u.ProjectID, u.TagName, opts, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("update release: %w", err)
		return
	}
	output = UpdateReleaseOutput{
		Success: true,
	}
	return
}
