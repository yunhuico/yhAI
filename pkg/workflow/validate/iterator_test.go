package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func TestIterator_Loop(t *testing.T) {
	t.Run("test empty nodes", func(t *testing.T) {
		iter := NewWorkflowNodeIter("startID", nil)
		err := iter.Loop(func(node model.Node) (end bool) {
			assert.False(t, true, "can not into this line")
			return
		})
		assert.Error(t, err)
	})

	t.Run("test start node not found", func(t *testing.T) {
		nodes := model.Nodes{
			{
				ID: "node2",
			},
		}
		iter := NewWorkflowNodeIter("startID", nodes)
		err := iter.Loop(func(node model.Node) (end bool) {
			assert.False(t, true, "can not into this line")
			return
		})
		assert.Error(t, err)
	})

	t.Run("test two linear nodes", func(t *testing.T) {
		nodes := model.Nodes{
			{
				ID: "node1",
				EditableNode: model.EditableNode{
					Transition: "node2",
				},
			},
			{
				ID: "node2",
			},
		}
		iter := NewWorkflowNodeIter("node1", nodes)
		count := 0
		err := iter.Loop(func(node model.Node) (end bool) {
			count++
			return
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, count)

		// just visit a node.
		count = 0
		err = iter.Loop(func(node model.Node) (end bool) {
			count++
			return true
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("test switch node", func(t *testing.T) {
		nodes := model.Nodes{
			{
				ID: "node1",
				EditableNode: model.EditableNode{
					Transition: "node2",
				},
			},
			{
				ID: "node2",
				EditableNode: model.EditableNode{
					Class: SwitchClass,
				},
				Data: model.NodeData{
					InputFields: map[string]any{
						"paths": []any{
							map[string]any{
								"name":       "path1",
								"transition": "node3",
							},
							map[string]any{
								"name":       "path2",
								"transition": "node5",
							},
							map[string]any{
								"name":       "path3-default",
								"transition": "node4",
								"isDefault":  true,
							},
						},
					},
				},
			},
			{
				ID: "node3",
			},
			{
				ID: "node4",
			},
			{
				ID: "node5",
			},
		}

		iter := NewWorkflowNodeIter("node1", nodes)
		count := 0
		err := iter.Loop(func(node model.Node) (end bool) {
			count++
			return
		})
		assert.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("test foreach node", func(t *testing.T) {
		iter := NewWorkflowNodeIter("node1", getForeachCaseNodes("node3"))
		count := 0
		err := iter.Loop(func(node model.Node) (end bool) {
			count++
			return
		})
		assert.NoError(t, err)
		assert.Equal(t, 4, count)
	})

	t.Run("test switch node has transition, will ignore the switch node.transition", func(t *testing.T) {
		nodes := model.Nodes{
			{
				ID: "node1",
				EditableNode: model.EditableNode{
					Transition: "node2",
				},
			},
			{
				ID: "node2",
				EditableNode: model.EditableNode{
					Class:      SwitchClass,
					Transition: "node3",
				},
			},
			{
				ID: "node3",
			},
		}

		iter := NewWorkflowNodeIter("node1", nodes)
		count := 0
		err := iter.Loop(func(node model.Node) (end bool) {
			count++
			return
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("test foreach node, foreach inner start node empty, no error", func(t *testing.T) {
		iter := NewWorkflowNodeIter("node1", getForeachCaseNodes(""))
		count := 0
		err := iter.Loop(func(node model.Node) (end bool) {
			count++
			return
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("test foreach node, foreach inner start node not found, assert error", func(t *testing.T) {
		iter := NewWorkflowNodeIter("node1", getForeachCaseNodes("not found"))
		count := 0
		err := iter.Loop(func(node model.Node) (end bool) {
			count++
			return
		})
		assert.Error(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("test two nodes cycled", func(t *testing.T) {
		nodes := model.Nodes{
			{
				ID: "node1",
				EditableNode: model.EditableNode{
					Transition: "node2",
				},
			},
			{
				ID: "node2",
				EditableNode: model.EditableNode{
					Transition: "node1",
				},
			},
		}
		iter := NewWorkflowNodeIter("node1", nodes)
		err := iter.Loop(func(node model.Node) (end bool) {
			assert.False(t, true, "can not into this line")
			return
		})
		assert.Error(t, err)
	})
}

func getForeachCaseNodes(innerTransition string) model.Nodes {
	return model.Nodes{
		{
			ID: "node1",
			EditableNode: model.EditableNode{
				Transition: "node2",
			},
		},
		{
			ID: "node2",
			EditableNode: model.EditableNode{
				Class:      ForeachClass,
				Transition: "node4",
			},
			Data: model.NodeData{
				InputFields: map[string]any{
					"transition": innerTransition,
				},
			},
		},
		{
			ID: "node3",
		},
		{
			ID: "node4",
		},
	}
}

func TestWorkflowNodeIter_GetDeleteNodeMaterials(t *testing.T) {
	t.Run("test empty nodes", func(t *testing.T) {
		iter := NewWorkflowNodeIter("startID", nil)
		_, err := iter.GetDeleteNodeMaterial("startID")
		assert.Error(t, err)
	})

	t.Run("test one normal node", func(t *testing.T) {
		assert := require.New(t)
		iter := NewWorkflowNodeIter("startID", model.Nodes{
			{
				ID: "startID",
			},
		})
		material, err := iter.GetDeleteNodeMaterial("startID")
		assert.NoError(err)
		assert.Equal("", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1)
	})

	t.Run("test switch node", func(t *testing.T) {
		assert := require.New(t)
		iter := NewWorkflowNodeIter("startNodeID", model.Nodes{
			{
				ID: "startNodeID",
				EditableNode: model.EditableNode{
					Transition: "node2",
				},
			},
			{
				ID: "node2",
				EditableNode: model.EditableNode{
					Class: SwitchClass,
				},
				Data: model.NodeData{
					InputFields: map[string]any{
						"paths": []any{
							map[string]any{
								"name":       "path1",
								"transition": "node3",
							},
							map[string]any{
								"name":       "path2",
								"transition": "node5",
							},
							map[string]any{
								"name":       "path3-default",
								"transition": "node4",
								"isDefault":  true,
							},
						},
					},
				},
			},
			{
				ID: "node3",
			},
			{
				ID: "node4",
			},
			{
				ID: "node5",
				EditableNode: model.EditableNode{
					Class: SwitchClass,
				},
				Data: model.NodeData{
					InputFields: map[string]any{
						"paths": []any{
							map[string]any{
								"name":       "default",
								"transition": "node6",
								"isDefault":  true,
							},
						},
					},
				},
			},
			{
				ID: "node6",
				EditableNode: model.EditableNode{
					Transition: "node7",
				},
			},
			{
				ID: "node7",
			},
		})

		material, err := iter.GetDeleteNodeMaterial("node2")
		assert.NoError(err)
		assert.Equal("startNodeID", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 6) // node2, node3, node4, node5, node6, node7

		material, err = iter.GetDeleteNodeMaterial("startNodeID")
		assert.NoError(err)
		assert.Equal("", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1) // just contains itself

		for _, nodeID := range []string{"node3", "node4"} {
			material, err = iter.GetDeleteNodeMaterial(nodeID)
			assert.NoError(err)
			assert.Equal("node2", material.PreviousNodeID)
			assert.Len(material.DeleteNodeIDChain, 1)
		}

		material, err = iter.GetDeleteNodeMaterial("node5")
		assert.NoError(err)
		assert.Equal("node2", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 3)

		material, err = iter.GetDeleteNodeMaterial("node6")
		assert.NoError(err)
		assert.Equal("node5", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1)

		material, err = iter.GetDeleteNodeMaterial("node7")
		assert.NoError(err)
		assert.Equal("node6", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1) // just contains itself
	})

	t.Run("test foreach node", func(t *testing.T) {
		assert := require.New(t)
		iter := NewWorkflowNodeIter("node1", getForeachCaseNodes("node3"))
		material, err := iter.GetDeleteNodeMaterial("node1")
		assert.NoError(err)
		assert.Equal("", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1)

		material, err = iter.GetDeleteNodeMaterial("node2")
		assert.NoError(err)
		assert.Equal("node1", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 2)

		material, err = iter.GetDeleteNodeMaterial("node3")
		assert.NoError(err)
		assert.Equal("node2", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1)

		material, err = iter.GetDeleteNodeMaterial("node4")
		assert.NoError(err)
		assert.Equal("node2", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1)
	})

	t.Run("test foreach + switch node", func(t *testing.T) {
		assert := require.New(t)
		iter := NewWorkflowNodeIter("startNodeID", model.Nodes{
			{
				ID: "startNodeID",
				EditableNode: model.EditableNode{
					Class:      ForeachClass,
					Transition: "node3",
				},
				Data: model.NodeData{
					InputFields: map[string]any{
						"transition": "node2",
					},
				},
			},
			{
				ID: "node2",
				EditableNode: model.EditableNode{
					Transition: "node4",
				},
			},
			{
				ID: "node3",
			},
			{
				ID: "node4",
				EditableNode: model.EditableNode{
					Class: SwitchClass,
				},
				Data: model.NodeData{
					InputFields: map[string]any{
						"paths": []any{
							map[string]any{
								"name":       "path1-default",
								"transition": "node5",
								"isDefault":  true,
							},
						},
					},
				},
			},
			{
				ID: "node5",
			},
		})

		material, err := iter.GetDeleteNodeMaterial("startNodeID")
		assert.NoError(err)
		assert.Equal("", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 4)

		material, err = iter.GetDeleteNodeMaterial("node4")
		assert.NoError(err)
		assert.Equal("node2", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 2)
	})

	t.Run("test empty foreach", func(t *testing.T) {
		assert := require.New(t)
		iter := NewWorkflowNodeIter("start", model.Nodes{
			{
				ID: "start",
				EditableNode: model.EditableNode{
					Class: ForeachClass,
				},
				Data: model.NodeData{
					InputFields: map[string]any{
						"transition": "",
					},
				},
			},
		})
		material, err := iter.GetDeleteNodeMaterial("start")
		assert.NoError(err)
		assert.Equal("", material.PreviousNodeID)
		assert.Len(material.DeleteNodeIDChain, 1)
	})
}
