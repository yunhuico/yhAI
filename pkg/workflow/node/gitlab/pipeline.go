package gitlab

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RunPipeline struct {
	BaseGitlabNode `json:"-"`

	ProjectID int        `json:"projectId"`
	Ref       *string    `json:"ref"`
	Variables []Variable `json:"variables"`
}

func (r RunPipeline) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/gitlab#runPipeline")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(RunPipeline)
		},
		InputForm: spec.InputSchema,
	}
}

func (r RunPipeline) getGitlabVariables() *[]*gitlab.PipelineVariable {
	if len(r.Variables) == 0 {
		return nil
	}
	varis := make([]*gitlab.PipelineVariable, 0, len(r.Variables))
	for _, variable := range r.Variables {
		if variable.Key == "" || variable.Value == "" {
			continue
		}
		varis = append(varis, &gitlab.PipelineVariable{
			Key:          variable.Key,
			Value:        variable.Value,
			VariableType: "env_var",
		})
	}
	return &varis
}

func (r RunPipeline) Run(c *workflow.NodeContext) (output any, err error) {
	opts := &gitlab.CreatePipelineOptions{
		Ref:       r.Ref,
		Variables: r.getGitlabVariables(),
	}
	pipeline, _, err := r.GetClient().Pipelines.CreatePipeline(r.ProjectID, opts, gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("create pipeline: %w", err)
		return
	}

	return pipeline, nil
}
