package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

func TestSwitchNode(t *testing.T) {
	log.Init("go-test", log.DebugLevel)
	switchNode := switchLogic{
		Paths: []validate.Path{
			{
				Name: "path 1", // id == 1
				Conditions: []validate.ConditionGroup{
					{
						{
							Left:      "{{ .Node.node1.id }}",
							Operation: validate.EqualsOperation,
							Right:     "1",
						},
					},
				},
				IsDefault: false,
			},
			{
				Name: "path 2", // id == 1 || id == 2
				Conditions: []validate.ConditionGroup{
					{
						{
							Left:      "{{ .Node.node1.id }}",
							Operation: validate.EqualsOperation,
							Right:     "1",
						},
					},
					{
						{
							Left:      "{{ .Node.node1.id }}",
							Operation: validate.EqualsOperation,
							Right:     "2",
						},
					},
				},
				IsDefault: false,
			},
			{
				Name:      "default path",
				IsDefault: true,
			},
		},
	}

	assert := require.New(t)
	w, err := NewWorkflowContext(context.Background(), ContextOpt{})
	assert.NoError(err)

	// test first branch matched.
	w.scope.setNodeData("node1", map[string]any{
		"id": 1,
	})
	result, err := switchNode.Run(w.newNodeContext(nil))
	assert.NoError(err)
	switchResults := result.(switchNodeResult)
	assert.Len(switchResults, 3)
	assert.Equal(switchResults[0].Name, "path 1")
	assert.Equal(switchResults[1].Name, "path 2")
	assert.Equal(switchResults[2].Name, "default path")
	assert.Equal(switchResults[0].ID, "1")
	assert.Equal(switchResults[1].ID, "2")
	assert.Equal(switchResults[2].ID, "default")
	assert.Equal(switchResults[0].ExecutionResult, true)
	assert.Equal(switchResults[1].ExecutionResult, false)
	assert.Equal(switchResults[2].ExecutionResult, false)

	// test middle branch matched.
	w.scope.setNodeData("node1", map[string]any{
		"id": 2,
	})
	result, err = switchNode.Run(w.newNodeContext(nil))
	assert.NoError(err)
	switchResults = result.(switchNodeResult)
	assert.Len(switchResults, 3)
	assert.Equal(switchResults[0].Name, "path 1")
	assert.Equal(switchResults[1].Name, "path 2")
	assert.Equal(switchResults[2].Name, "default path")
	assert.Equal(switchResults[0].ID, "1")
	assert.Equal(switchResults[1].ID, "2")
	assert.Equal(switchResults[2].ID, "default")
	assert.Equal(switchResults[0].ExecutionResult, false)
	assert.Equal(switchResults[1].ExecutionResult, true)
	assert.Equal(switchResults[2].ExecutionResult, false)

	// test default branch matched.
	w.scope.setNodeData("node1", map[string]any{
		"id": 3,
	})
	result, err = switchNode.Run(w.newNodeContext(nil))
	assert.NoError(err)
	switchResults = result.(switchNodeResult)
	assert.Len(switchResults, 3)
	assert.Equal(switchResults[0].Name, "path 1")
	assert.Equal(switchResults[1].Name, "path 2")
	assert.Equal(switchResults[2].Name, "default path")
	assert.Equal(switchResults[0].ID, "1")
	assert.Equal(switchResults[1].ID, "2")
	assert.Equal(switchResults[2].ID, "default")
	assert.Equal(switchResults[0].ExecutionResult, false)
	assert.Equal(switchResults[1].ExecutionResult, false)
	assert.Equal(switchResults[2].ExecutionResult, true)
}
