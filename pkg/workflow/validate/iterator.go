package validate

import (
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
)

type WorkflowNodeIter struct {
	startNodeID string
	nodes       model.Nodes
	nodesMap    map[string]model.Node
}

func NewWorkflowNodeIter(startNodeID string, nodes model.Nodes) *WorkflowNodeIter {
	return &WorkflowNodeIter{
		startNodeID: startNodeID,
		nodes:       nodes,
		nodesMap:    nodes.MapByID(),
	}
}

// Loop through the node chain from the start node.
//
// the loop will stop if transition not found in the node maps.
// for switch-node, visit nodes path by path, visit defaultTransition in the end.
// for foreach-node, visit inner nodes first, then visit outer transition node.
func (it *WorkflowNodeIter) Loop(fn func(node model.Node) (end bool)) (err error) {
	report := &SynthesisReport{}
	validateNodeDAG(report, it.nodes)
	if report.ExistsFatal() {
		return report
	}

	var nextNodeTransitionStack []string
	curNodeID := it.startNodeID
	for {
		if curNodeID == "" {
			if len(nextNodeTransitionStack) == 0 {
				break
			}

			// pop the next node from the stack.
			curNodeID = nextNodeTransitionStack[0]
			nextNodeTransitionStack = nextNodeTransitionStack[1:]
		}

		node := it.nodesMap[curNodeID]
		if node.ID == "" {
			err = fmt.Errorf("node %s not found", curNodeID)
			break
		}

		end := fn(node)
		if end {
			return
		}

		switch node.Class {
		case ForeachClass:
			if node.Transition != "" {
				nextNodeTransitionStack = append(nextNodeTransitionStack, node.Transition)
			}
			curNodeID, _ = node.GetForeachStartNodeID()
		case SwitchClass:
			var switchNode SwitchLogicNode
			err = trans.MapToStruct(node.Data.InputFields, &switchNode)
			if err != nil {
				err = fmt.Errorf("switch input field error: %s", err)
				return
			}

			curNodeID = ""
			for i := 0; i < len(switchNode.Paths); i++ {
				if switchNode.Paths[i].Transition != "" {
					nextNodeTransitionStack = append(nextNodeTransitionStack, switchNode.Paths[i].Transition)
				}
			}
		default:
			curNodeID = node.Transition
		}
	}

	return
}

type DeleteNodeMaterial struct {
	Node              model.Node
	PreviousNodeID    string
	PreviousNode      model.Node
	DeleteNodeIDChain []string // contains delete targetID itself, so this slice has at least one node.
}

// GetDeleteNodeMaterial we promise each node only has one previous node transition to it.
// if deleteTargetNode is a switch or foreach node, should delete the whole node-unit(contains every node in foreach or in switch.)
// TODO(sword): consider confirm node.
func (it *WorkflowNodeIter) GetDeleteNodeMaterial(deleteTargetNodeID string) (material DeleteNodeMaterial, err error) {
	var ok bool
	report := &SynthesisReport{}
	validateNodeDAG(report, it.nodes)
	if report.ExistsFatal() {
		err = report
		return
	}

	material.Node, ok = it.nodesMap[deleteTargetNodeID]
	if !ok {
		err = fmt.Errorf("iterNode %s not found", deleteTargetNodeID)
		return
	}

	// find its previous iterNode.
	for _, iterNode := range it.nodes {
		if iterNode.Transition == deleteTargetNodeID {
			material.PreviousNode = iterNode
			material.PreviousNodeID = iterNode.ID
			break
		}
	}

	for _, iterNode := range it.nodes {
		// if material.PreviousNodeID == "", indicate that the previous iterNode maybe is switchNode or foreachNode.
		if material.PreviousNodeID != "" {
			break
		}
		if iterNode.Class == SwitchClass {
			var switchNode SwitchLogicNode
			err = trans.MapToStruct(iterNode.Data.InputFields, &switchNode)
			if err != nil {
				err = fmt.Errorf("switch input field error: %s", err)
				return
			}
			for _, path := range switchNode.Paths {
				if path.Transition == deleteTargetNodeID {
					material.PreviousNode = iterNode
					material.PreviousNodeID = iterNode.ID
					break
				}
			}
		} else if iterNode.Class == ForeachClass {
			if foreachNodeStartID, _ := iterNode.GetForeachStartNodeID(); foreachNodeStartID == deleteTargetNodeID {
				material.PreviousNode = iterNode
				material.PreviousNodeID = iterNode.ID
			}
		}
	}

	var nodeIDList []string
	err = it.lookThroughNode(&nodeIDList, material.Node)
	if err != nil {
		err = fmt.Errorf("deep into iterNode: %w", err)
	}

	material.DeleteNodeIDChain = append(nodeIDList, deleteTargetNodeID)

	return
}

// lookThroughNode will look inside node and find all related nodes
func (it *WorkflowNodeIter) lookThroughNode(nodeIDList *[]string, node model.Node) (err error) {
	var throughStartID string
	switch node.Class {
	case SwitchClass:
		var switchNode SwitchLogicNode
		err = trans.MapToStruct(node.Data.InputFields, &switchNode)
		if err != nil {
			err = fmt.Errorf("switch input field error: %s", err)
			return
		}
		for _, path := range switchNode.Paths {
			if path.Transition == "" {
				continue
			}
			err = it.collectNodeFromChain(nodeIDList, path.Transition, node.Transition)
			if err != nil {
				err = fmt.Errorf("collect node through %q to %q error: %w", path.Transition, node.Transition, err)
				return
			}
		}
	case ForeachClass:
		var ok bool
		throughStartID, ok = node.GetForeachStartNodeID()
		if !ok {
			return
		}
	}

	err = it.collectNodeFromChain(nodeIDList, throughStartID, node.Transition)
	if err != nil {
		err = fmt.Errorf("collect node through %q to %q error: %w", throughStartID, node.Transition, err)
		return
	}

	return
}

// collectNodeFromChain will collect all nodes from startNodeID until endNodeID (node with endNodeID will not be collected)
func (it *WorkflowNodeIter) collectNodeFromChain(nodeIDList *[]string, startNodeID string, endNodeID string) (err error) {
	currentNodeID := startNodeID
	for currentNodeID != "" && currentNodeID != endNodeID {
		*nodeIDList = append(*nodeIDList, currentNodeID)

		err = it.lookThroughNode(nodeIDList, it.nodesMap[currentNodeID])
		if err != nil {
			err = fmt.Errorf("deep into node: %w", err)
			return
		}

		currentNodeID = it.nodesMap[currentNodeID].Transition
	}
	return
}
