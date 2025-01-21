package apiserver

import (
	"context"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

var (
	errSwitchNotAtTail      error = fmt.Errorf("switch node can not be follewed by other nodes: %w", errCreateNodeDenied)
	errOutOfSwitchPathBound       = fmt.Errorf("switch path index out of bound: %w", errCreateNodeDenied)
)

// transitionHandler helps to handle the process of creating a new node
// setRelatedNodesTransition will set related nodes' transitions, depending on the position of the new node
// checkSwitchAtTail makes sure switch is at the tail of a chain of nodes(i.e., there is no node behind switch)
type transitionHandler interface {
	setRelatedNodesTransition() error
	checkSwitchAtTail() error
}

type normalHandler struct {
	previousNode *model.Node
	newNode      *model.Node
}

func newNormalHandler(previousNode *model.Node, newNode *model.Node) *normalHandler {
	return &normalHandler{
		previousNode: previousNode,
		newNode:      newNode,
	}
}

func (h *normalHandler) setRelatedNodesTransition() (err error) {
	h.newNode.Transition = h.previousNode.Transition
	h.previousNode.Transition = h.newNode.ID

	return
}

func (h *normalHandler) checkSwitchAtTail() error {
	if h.newNode.Class == validate.SwitchClass && h.previousNode.Transition != "" {
		return errSwitchNotAtTail
	}

	return nil
}

type firstInsideForeachHandler struct {
	normalHandler

	previousForeachStruct validate.LoopFromListNode
}

func newFirstInsideForeachHandler(previousNode *model.Node, newNode *model.Node) (handler *firstInsideForeachHandler, err error) {
	handler = &firstInsideForeachHandler{
		normalHandler: normalHandler{
			previousNode: previousNode,
			newNode:      newNode,
		},
	}
	err = trans.MapToStruct(previousNode.Data.InputFields, &handler.previousForeachStruct)
	if err != nil {
		err = fmt.Errorf("foreach node input field error: %w", err)
		return nil, err
	}

	return
}

func (h *firstInsideForeachHandler) setRelatedNodesTransition() (err error) {
	h.newNode.Transition = h.previousForeachStruct.Transition
	h.previousForeachStruct.Transition = h.newNode.ID

	var forEachNodeMap map[string]any
	forEachNodeMap, err = trans.StructToMap(h.previousForeachStruct)
	if err != nil {
		err = fmt.Errorf("foreach node struct error: %w", err)
		return
	}
	h.previousNode.Data.InputFields = forEachNodeMap
	return
}

func (h *firstInsideForeachHandler) checkSwitchAtTail() (err error) {
	if h.newNode.Transition == validate.SwitchClass && h.previousForeachStruct.Transition != "" {
		return errSwitchNotAtTail
	}

	return
}

type firstInsideSwitchHandler struct {
	normalHandler

	previousSwitchStruct validate.SwitchLogicNode
	switchPathIndex      int
}

func newFirstInsideSwitchHandler(previousNode *model.Node, newNode *model.Node, switchPathIndex int) (handler *firstInsideSwitchHandler, err error) {
	handler = &firstInsideSwitchHandler{
		normalHandler: normalHandler{
			previousNode: previousNode,
			newNode:      newNode,
		},
		switchPathIndex: switchPathIndex,
	}

	err = trans.MapToStruct(previousNode.Data.InputFields, &handler.previousSwitchStruct)
	if err != nil {
		err = fmt.Errorf("switch node input field error: %w", err)
		return
	}
	if switchPathIndex > len(handler.previousSwitchStruct.Paths) || switchPathIndex < -1 {
		err = errOutOfSwitchPathBound
		return nil, err
	}

	return
}

func (h *firstInsideSwitchHandler) setRelatedNodesTransition() (err error) {
	h.newNode.Transition = h.previousSwitchStruct.Paths[h.switchPathIndex].Transition
	h.previousSwitchStruct.Paths[h.switchPathIndex].Transition = h.newNode.ID

	var switchNodeMap map[string]any
	switchNodeMap, err = trans.StructToMap(h.previousSwitchStruct)
	if err != nil {
		err = fmt.Errorf("foreach node struct error: %w", err)
		return
	}
	h.previousNode.Data.InputFields = switchNodeMap

	return
}

func (h *firstInsideSwitchHandler) checkSwitchAtTail() error {
	oldTransition := h.previousSwitchStruct.Paths[h.switchPathIndex].Transition

	if h.newNode.Class == validate.SwitchClass && oldTransition != "" {
		return errSwitchNotAtTail
	}

	return nil
}

type CreateNodeProcess struct {
	payload.PreviousNodeInfo

	ctx          context.Context
	previousNode *model.Node
	newNode      *model.Node
	db           *model.DB
}

func newCreateNodeProcess(ctx context.Context, newNode *model.Node, previousNodeInfo payload.PreviousNodeInfo, db *model.DB) (process *CreateNodeProcess, err error) {
	previousNode, err := db.GetNodeByID(ctx, previousNodeInfo.PreviousNodeID)
	if err != nil {
		err = fmt.Errorf("querying for the previous node: %w", err)
		return
	}

	process = &CreateNodeProcess{
		PreviousNodeInfo: previousNodeInfo,
		ctx:              ctx,
		previousNode:     &previousNode,
		newNode:          newNode,
		db:               db,
	}

	return

}

// execute will create a new node in the workflow
//  1. insert the new node into database
//  2. if the new node is a switch node, make true it is at the tail of a chain of nodes
//  3. set transitions of related nodes
//     3.1 new node is after a node
//     3.2 new node is the first node inside a foreach node
//     3.3 new node is the first node inside a switch node
//     3.4 TODO(yuhao): new node is the first node inside a confirm node
//  4. update database
func (process *CreateNodeProcess) execute() (err error) {
	var ctx = process.ctx
	err = process.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		// 1.
		err = tx.InsertNode(ctx, process.newNode)
		if err != nil {
			return
		}

		var handler transitionHandler
		if !process.IsFirstInsideNode {
			// 3.1
			handler = newNormalHandler(process.previousNode, process.newNode)
		} else {
			switch process.previousNode.Class {
			case validate.ForeachClass: // 3.2
				handler, err = newFirstInsideForeachHandler(process.previousNode, process.newNode)
				if err != nil {
					return
				}
			case validate.SwitchClass: // 3.3
				handler, err = newFirstInsideSwitchHandler(process.previousNode, process.newNode, process.PreviousSwitchPathIndex)
				if err != nil {
					return
				}
			}
			// TODO(yuhao): confirmNode 3.4
		}

		// 2.
		err = handler.checkSwitchAtTail()
		if err != nil {
			return fmt.Errorf("check switch at tail error: %w", err)
		}

		// 3.
		err = handler.setRelatedNodesTransition()
		if err != nil {
			return fmt.Errorf("set related nodes' transition error: %w", err)
		}

		// 4.
		err = tx.UpdateNodeByID(ctx, process.previousNode)
		if err != nil {
			return fmt.Errorf("update node error: %w", err)
		}

		err = tx.UpdateNodeTransition(ctx, process.newNode.ID, process.newNode.Transition)
		if err != nil {
			return fmt.Errorf("update node transition error: %w", err)
		}
		return
	})

	return
}
