package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	_ "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

func (s *testServer) updateSwitchNodePathName(workflowID, nodeID string, p *payload.UpdateSwitchNodePathNameReq, assertFns ...assertOptFn) {
	opt := getAssertOpt(assertFns)
	b, err := json.Marshal(p)
	s.NoError(err)
	resp := s.request("PUT", fmt.Sprintf("/api/v1/workflows/%s/nodes/%s/pathName", workflowID, nodeID), bytes.NewReader(b))
	s.Equal(opt.httpStatusCode, resp.Code, fmt.Sprintf("body: %s", resp.Body.String()))

	r := &R{}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	s.Equal(opt.errorCode, r.Code)
}

func assertNodeExist(t *testing.T, nodeID string, db *model.DB) (node model.Node) {
	node, err := db.GetNodeByID(ctx, nodeID)
	assert.NoError(t, err)
	return
}

func assertNoNode(t *testing.T, nodeID string, db *model.DB) {
	_, err := db.GetNodeByID(ctx, nodeID)
	assert.Error(t, err)
}

func assertNodeTransitionIs(t *testing.T, nodeID, transitionNodeID string, db *model.DB) {
	node, err := db.GetNodeByID(ctx, nodeID)
	assert.NoError(t, err)
	assert.Equal(t, transitionNodeID, node.Transition)
}

/*
*

	              +-----------+
	              | startNode |
	              +-----------+
	                    |
	                    v
	+------------------------------------------+
	|              forEachNode                 |
	|           +----------------+             |
	|           |   switchNode   |             |
	|           +----------------+             |
	|                    |                     |
	|                    |                     |
	|      -----------------------------       |
	|      |             |             |       |
	|      v             v             v       |
	|+-----------+                             |
	||debugNode1 |                             |
	|+-----------+                             |
	|+-----------+                             |
	||debugNode2 |                             |
	|+-----------+                             |
	|+-----------+                             |
	||debugNode3 |                             |
	|+-----------+                             |
	+------------------------------------------+
	                    |
	                    v
	             +--------------+
	             | forEachNode2 |
	             | +----------+ |
	             | |debugNode4| |
	             | +----------+ |
	             +--------------+
	                    |
	                    v
	             +-------------+
	             | switchNode2 |
	             +-------------+
	                    |
	                    v
	            -----------------
	            |               |
	            v               v
	     +--------------+
	     | forEachNode3 |
	     |+-----------+ |
	     ||debugNode5 | |
	     |+-----------+ |
	     |+-----------+ |
	     ||debugNode6 | |
	     |+-----------+ |
	     +--------------+
	      +-----------+
	      |debugNode7 |
	      +-----------+
	      +-----------+
	      |debugNode8 |
	      +-----------+
*/
func TestDeleteNode(t *testing.T) {
	server := newTestServer(t)
	db := server.db
	workflowID := server.createWorkflow(&payload.EditWorkflowReq{
		Name: "testWorkflowForDeleteNode",
	})
	err := db.InsertNode(ctx, &model.Node{
		ID: "startNode",
		EditableNode: model.EditableNode{
			Name:       "startNode",
			Class:      validate.CronTriggerClass,
			Transition: "forEachNode",
		},
		WorkflowID: workflowID,
		Type:       "trigger",
	})
	assert.NoError(t, err)
	err = db.InsertNode(ctx, &model.Node{
		ID: "forEachNode",
		EditableNode: model.EditableNode{
			Name:       "forEachNode",
			Class:      validate.ForeachClass,
			Transition: "forEachNode2",
		},
		Type: "logic",
		Data: model.NodeData{
			InputFields: map[string]any{
				"inputCollection": "foreach",
				"transition":      "switchNode",
			},
		},
		WorkflowID: workflowID,
	})
	assert.NoError(t, err)
	err = db.InsertNode(ctx, &model.Node{
		ID: "switchNode",
		EditableNode: model.EditableNode{
			Name:  "switchNode",
			Class: validate.SwitchClass,
		},
		Type: "logic",
		Data: model.NodeData{
			InputFields: map[string]any{
				"paths": []map[string]any{
					{"name": "path1", "transition": "debugNode1"},
					{"name": "path2", "transition": ""},
					{"name": "path3", "transition": ""},
				},
			},
		},
		WorkflowID: workflowID,
	})
	assert.NoError(t, err)
	err = db.InsertNode(ctx, &model.Node{
		ID: "forEachNode2",
		EditableNode: model.EditableNode{
			Name:       "forEachNode2",
			Class:      validate.ForeachClass,
			Transition: "switchNode2",
		},
		Type: "logic",
		Data: model.NodeData{
			InputFields: map[string]any{
				"inputCollection": "foreach",
				"transition":      "debugNode4",
			},
		},
		WorkflowID: workflowID,
	})
	assert.NoError(t, err)
	err = db.InsertNode(ctx, &model.Node{
		ID: "switchNode2",
		EditableNode: model.EditableNode{
			Name:  "switchNode2",
			Class: validate.SwitchClass,
		},
		Type: "logic",
		Data: model.NodeData{
			InputFields: map[string]any{
				"paths": []map[string]any{
					{"name": "path1", "transition": "forEachNode3"},
					{"name": "path2", "transition": ""},
				},
			},
		},
		WorkflowID: workflowID,
	})
	assert.NoError(t, err)
	err = db.InsertNode(ctx, &model.Node{
		ID: "forEachNode3",
		EditableNode: model.EditableNode{
			Name:       "forEachNode3",
			Class:      validate.ForeachClass,
			Transition: "debugNode7",
		},
		Type: "logic",
		Data: model.NodeData{
			InputFields: map[string]any{
				"inputCollection": "foreach",
				"transition":      "debugNode5",
			},
		},
		WorkflowID: workflowID,
	})
	assert.NoError(t, err)

	generateDebugNodeChain := func(startID, endID int) {
		for nodeNum := startID; nodeNum < endID; nodeNum++ {
			nodeID := fmt.Sprintf("debugNode%d", nodeNum)
			nextNodeID := fmt.Sprintf("debugNode%d", nodeNum+1)
			err = db.InsertNode(ctx, &model.Node{
				ID: nodeID,
				EditableNode: model.EditableNode{
					Name:       nodeID,
					Class:      "ultrafox/debug#printTarget",
					Transition: nextNodeID,
				},
				Type:       "actor",
				WorkflowID: workflowID,
			})
			assert.NoError(t, err)
		}
		endNodeID := fmt.Sprintf("debugNode%d", endID)
		err = db.InsertNode(ctx, &model.Node{
			ID: endNodeID,
			EditableNode: model.EditableNode{
				Name:       endNodeID,
				Class:      "ultrafox/debug#printTarget",
				Transition: "",
			},
			Type:       "actor",
			WorkflowID: workflowID,
		})
	}
	generateDebugNodeChain(1, 3)
	generateDebugNodeChain(4, 4)
	generateDebugNodeChain(5, 6)
	generateDebugNodeChain(7, 8)

	t.Run("delete a normal node", func(t *testing.T) {
		server.deleteNode(workflowID, "debugNode2", assertErrorCode(0))
		assertNoNode(t, "debugNode2", db)
		assertNodeTransitionIs(t, "debugNode1", "debugNode3", db)
	})

	t.Run("delete the first child node of a switch node", func(t *testing.T) {
		server.deleteNode(workflowID, "debugNode1", assertErrorCode(0))
		assertNoNode(t, "debugNode1", db)
		switchNode, err := db.GetNodeByID(ctx, "switchNode")
		assert.NoError(t, err)
		var switchNodeStruct validate.SwitchLogicNode
		err = trans.MapToStruct(switchNode.Data.InputFields, &switchNodeStruct)
		assert.NoError(t, err)
		assert.Equal(t, "debugNode3", switchNodeStruct.Paths[0].Transition)
	})

	t.Run("delete a non empty foreach node", func(t *testing.T) {
		server.deleteNode(workflowID, "forEachNode")
		assertNoNode(t, "forEachNode", db)
		assertNoNode(t, "switchNode", db)
		assertNoNode(t, "debugNode3", db)
		assertNodeTransitionIs(t, "startNode", "forEachNode2", db)
	})

	t.Run("delete the fist child node of a foreach node", func(t *testing.T) {
		server.deleteNode(workflowID, "debugNode4")
		assertNoNode(t, "debugNode4", db)
		forEachNode2, err := db.GetNodeByID(ctx, "forEachNode2")
		assert.NoError(t, err)
		var forEachNodeStruct validate.LoopFromListNode
		err = trans.MapToStruct(forEachNode2.Data.InputFields, &forEachNodeStruct)
		assert.NoError(t, err)
		assert.Equal(t, "", forEachNodeStruct.Transition)
	})

	t.Run("delete an empty foreach node", func(t *testing.T) {
		server.deleteNode(workflowID, "forEachNode2")
		assertNoNode(t, "forEachNode2", db)
		assertNodeTransitionIs(t, "startNode", "switchNode2", db)
	})

	t.Run("delete the first node after a foreach node", func(t *testing.T) {
		server.deleteNode(workflowID, "debugNode7")
		assertNoNode(t, "debugNode7", db)
		forEachNode3, err := db.GetNodeByID(ctx, "forEachNode3")
		assert.NoError(t, err)

		// first node inside the foreach node should remain unchanged
		var forEachNodeStruct validate.LoopFromListNode
		err = trans.MapToStruct(forEachNode3.Data.InputFields, &forEachNodeStruct)
		assert.NoError(t, err)
		assert.Equal(t, "debugNode5", forEachNodeStruct.Transition)
	})

	t.Run("delete non-empty switch node", func(t *testing.T) {
		server.deleteNode(workflowID, "switchNode2")
		assertNoNode(t, "switchNode2", db)
		for i := 5; i <= 8; i++ {
			assertNoNode(t, fmt.Sprintf("debugNode%d", i), db)
		}
		assertNoNode(t, "forEachNode3", db)
		assertNodeTransitionIs(t, "startNode", "", db)
	})
}

func TestCreateNode(t *testing.T) {
	server := newTestServer(t)
	db := server.db
	workflowID := server.createWorkflow(&payload.EditWorkflowReq{
		Name: "testCreateWorkflowNode",
	})

	// create start node
	startNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "startNode",
			Class: validate.CronTriggerClass,
		},
		IsStart: true,
		InputFields: map[string]any{
			"expr":     "* * * * * *",
			"timezone": "Asia/Shanghai",
		},
	})
	assertNodeExist(t, startNodeID, db)

	// create a normal node
	debugNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "debugNode",
			Class: "ultrafox/debug#printTarget",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID: startNodeID,
		},
		InputFields: map[string]any{
			"target": "foo",
		},
	})
	assertNodeExist(t, debugNodeID, db)
	assertNodeTransitionIs(t, startNodeID, debugNodeID, db)

	// create a foreach node
	foreachNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "foreachNode",
			Class: validate.ForeachClass,
		},
		InputFields: map[string]any{
			"inputCollection": "foo",
			"transition":      "",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID: debugNodeID,
		},
	})
	assertNodeExist(t, foreachNodeID, db)
	assertNodeTransitionIs(t, debugNodeID, foreachNodeID, db)

	// create first child node of foreach node
	debugNodeID2 := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "debugNode2",
			Class: "ultrafox/debug#printTarget",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			IsFirstInsideNode: true,
			PreviousNodeID:    foreachNodeID,
		},
		InputFields: map[string]any{
			"target": "foo",
		},
	})
	assertNodeExist(t, debugNodeID2, db)
	var forEachNodeStruct validate.LoopFromListNode
	foreachNode, err := db.GetNodeByID(ctx, foreachNodeID)
	assert.NoError(t, err)
	err = trans.MapToStruct(foreachNode.Data.InputFields, &forEachNodeStruct)
	assert.NoError(t, err)
	assert.Equal(t, debugNodeID2, forEachNodeStruct.Transition)

	// create a switch node
	switchNodeID := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "switchNode",
			Class: validate.SwitchClass,
		},
		InputFields: map[string]any{
			"paths": []any{
				map[string]any{
					"name":       "path",
					"conditions": []any{},
					"transition": "",
				},
			},
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID: foreachNodeID,
		},
	})
	assertNodeExist(t, switchNodeID, db)
	assertNodeTransitionIs(t, foreachNodeID, switchNodeID, db)

	// update switch path name
	server.updateSwitchNodePathName(workflowID, switchNodeID, &payload.UpdateSwitchNodePathNameReq{
		Index: 0,
		Name:  "Path2",
	})
	// update a not-switch node path name
	server.updateSwitchNodePathName(workflowID, debugNodeID2, &payload.UpdateSwitchNodePathNameReq{
		Index: 0,
		Name:  "Path2",
	}, assertHTTPStatusCode(400), assertErrorCode(600402))

	// create the first node of switch node
	debugNodeID3 := server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "debugNode2",
			Class: "ultrafox/debug#printTarget",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			IsFirstInsideNode:       true,
			PreviousSwitchPathIndex: 0,
			PreviousNodeID:          switchNodeID,
		},
		InputFields: map[string]any{
			"target": "foo",
		},
	})
	assertNodeExist(t, switchNodeID, db)
	switchNode, err := db.GetNodeByID(ctx, switchNodeID)
	assert.NoError(t, err)
	var switchNodeStruct validate.SwitchLogicNode
	err = trans.MapToStruct(switchNode.Data.InputFields, &switchNodeStruct)
	assert.NoError(t, err)
	assert.Equal(t, debugNodeID3, switchNodeStruct.Paths[0].Transition)
	assert.Equal(t, "Path2", switchNodeStruct.Paths[0].Name)

	// create a switch node before a node
	_ = server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "notValidSwitchNode",
			Class: validate.SwitchClass,
		},
		InputFields: map[string]any{
			"paths": []any{
				map[string]any{
					"name":       "path",
					"conditions": []any{},
					"transition": "",
				},
			},
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			PreviousNodeID: startNodeID,
		},
	}, assertErrorCode(codeInternalError))

	// create a node out of the bound of switch node's path
	_ = server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "invalidDebugNodeInsideSwitch",
			Class: "ultrafox/debug#printTarget",
		},
		PreviousNodeInfo: payload.PreviousNodeInfo{
			IsFirstInsideNode: true,
			// currently, switchNode has only 1 path
			PreviousSwitchPathIndex: 5,
			PreviousNodeID:          switchNodeID,
		},
		InputFields: map[string]any{
			"target": "foo",
		},
	}, assertErrorCode(codeInternalError))
}

func TestNodeExtFields(t *testing.T) {
	server := newTestServer(t)
	workflowID := server.createWorkflow(&payload.EditWorkflowReq{
		Name: "testCreateWorkflowNode",
	})
	_ = server.createNode(workflowID, &payload.EditNodeReq{
		EditableNode: model.EditableNode{
			Name:  "startNode",
			Class: validate.CronTriggerClass,
		},
		IsStart: true,
		InputFields: map[string]any{
			"expr":     "* * * * * *",
			"timezone": "Asia/Shanghai",
		},
		ExtFields: map[string]any{
			"timezone": map[string]any{
				"isCustomInput": false,
				"selectedId":    "tz-1",
			},
		},
	})
	workflow := server.getWorkflow(workflowID)
	assert.NotNil(t, workflow)
	assert.Len(t, workflow.Workflow.Nodes, 1)
	assert.Equal(t, map[string]any{
		"timezone": map[string]any{
			"isCustomInput": false,
			"selectedId":    "tz-1",
		},
	}, workflow.Workflow.Nodes[0].Data.ExtFields)
}
