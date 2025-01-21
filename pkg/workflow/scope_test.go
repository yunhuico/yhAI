package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScopeDataToNodePresentMap(t *testing.T) {
	t.Run("test a valid data", func(t *testing.T) {
		scope := newContextScope()
		scope.setNodeData("node1", map[string]any{
			"input": map[string]any{
				"id":   1,
				"name": "ultrafox",
			},
			"output": map[string]any{
				"success": true,
			},
		})

		scope.setNodeData("node2", map[string]any{
			"input": map[string]any{
				"id":   2,
				"site": "ultrafox.io",
			},
			"output": map[string]any{
				"content": "ultrafox awesome",
			},
		})
	})
}

// TestThreeLevelScopeData monitor confirm data structure.
// confirm data scope using contextScopeProxy. if a confirm-node in a foreach loop,
// then contextScopeProxy will clone a contextScopeProxy again.
func TestThreeLevelScopeData(t *testing.T) {
	scope := newContextScope()
	scope.setNodeData("node1", map[string]any{
		"input": map[string]any{
			"id":   1,
			"name": "ultrafox",
		},
		"output": map[string]any{
			"success": true,
		},
	})
	scope2 := scope.cloneProxy()

	scope2.setNodeData("confirmNode1", map[string]any{
		"output": map[string]any{
			"success": true,
		},
	})
	scope2.setNodeData("confirmNode2", map[string]any{
		"output": map[string]any{
			"success": true,
		},
	})

	scope3 := scope2.cloneProxy()
	scope3.setNodeData("foreachNode1", map[string]any{
		"output": map[string]any{
			"success": true,
		},
	})

	scope3Data := scope3.getDiffNodeData()
	assert.Len(t, scope3Data, 1)

	scope2Data := scope2.getDiffNodeData()
	assert.Len(t, scope2Data, 3)
}
