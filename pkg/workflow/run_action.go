package workflow

import (
	"context"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
)

// BaseWorkflowActionOpt all workflow action require these fields.
type BaseWorkflowActionOpt struct {
	Ctx                  context.Context
	DB                   *model.DB
	Cache                *cache.Cache
	WorkflowWithNodes    model.WorkflowWithNodesCredential
	Cipher               crypto.CryptoCipher
	ServerHost           *serverhost.ServerHost
	MailSender           *smtp.Sender
	PassportVendorLookup map[model.PassportVendorName]model.PassportVendor
}

type TestWorkflowActionOpt struct {
	BaseWorkflowActionOpt

	NodeID      string
	ContextData map[string]any
	ForeachNode *model.Node
	IterIndex   int
}

type TestWorkflowAction struct {
	workflowContext *WorkflowContext
	iterIndex       int
	foreachNode     *model.Node
	nodeID          string
}

func NewTestWorkflowAction(opt TestWorkflowActionOpt) (action *TestWorkflowAction, err error) {
	contextOpt := ContextOpt{
		DB:                   opt.DB,
		WorkflowWithNodes:    opt.WorkflowWithNodes,
		Cipher:               opt.Cipher,
		ServerHost:           opt.ServerHost,
		MailSender:           opt.MailSender,
		testRunNodeID:        opt.NodeID,
		Cache:                opt.Cache,
		PassportVendorLookup: opt.PassportVendorLookup,
	}
	if opt.ForeachNode != nil {
		contextOpt.testForeachNodeID = opt.ForeachNode.ID
	}
	workflowContext, err := NewTestWorkflowContext(opt.Ctx, contextOpt, opt.ContextData)

	action = &TestWorkflowAction{
		nodeID:          opt.NodeID,
		foreachNode:     opt.ForeachNode,
		iterIndex:       opt.IterIndex,
		workflowContext: workflowContext,
	}

	return
}

func (a *TestWorkflowAction) GetWorkflowContext() *WorkflowContext {
	return a.workflowContext
}

func (a *TestWorkflowAction) Run() error {
	if a.foreachNode != nil {
		// change the foreach node output, just keep one item.
		inputCollectionAny, ok := a.foreachNode.Data.InputFields["inputCollection"]
		if !ok {
			return errors.New("foreach inputCollection not defined")
		}
		inputCollectionExpression, _ := trimBraceBrackets(inputCollectionAny.(string))
		collection, err := a.workflowContext.getIteration(inputCollectionExpression)
		if err != nil {
			return fmt.Errorf("getting foreach iteration: %w", err)
		}
		if a.iterIndex >= len(collection) {
			return fmt.Errorf("iterIndex exceed max")
		}

		a.workflowContext.setIterKeyValue(iterLengthKey, len(collection))
		a.workflowContext.setCurrentIteration(a.iterIndex+1, collection[a.iterIndex], a.iterIndex+1 == len(collection))
	}

	return a.workflowContext.Run(a.nodeID, nil).Err
}

func (a *TestWorkflowAction) RunStartNode(input any) error {
	a.workflowContext.useExternalInput = true // run first node, the input should pass by client.
	return a.workflowContext.Run(a.nodeID, input).Err
}

type RunNodeInstanceActionOpt struct {
	BaseWorkflowActionOpt

	CredentialID string
	Spec         *adapter.Spec
	InputFields  map[string]any
	Credential   *model.CredentialWithParent
}

// QueryFieldResultAction not require context-dependent int the current phase,
// add context parameters if needed.
type QueryFieldResultAction struct {
	node            model.NodeWithCredential
	workflowContext *WorkflowContext
}

func (a QueryFieldResultAction) Run() (result QueryFieldResult, err error) {
	nodeIns, err := NewNodeInstance(a.workflowContext.Context(), a.workflowContext.db, a.workflowContext.cache, a.workflowContext.cipher, a.node)
	if err != nil {
		err = fmt.Errorf("new node instance: %w", err)
		return
	}

	err = a.workflowContext.BindInputFieldsToNode(nodeIns)
	if err != nil {
		err = fmt.Errorf("unmarshal node from raw parameters: %w", err)
		return
	}

	result, err = a.workflowContext.queryFieldResultList(nodeIns)
	return
}

func NewQueryFieldResultAction(opt RunNodeInstanceActionOpt) (action QueryFieldResultAction, err error) {
	workflowContext, err := NewWorkflowContext(opt.Ctx, ContextOpt{
		DB:                   opt.DB,
		Cache:                opt.Cache,
		WorkflowWithNodes:    opt.WorkflowWithNodes,
		Cipher:               opt.Cipher,
		ServerHost:           opt.ServerHost,
		MailSender:           opt.MailSender,
		PassportVendorLookup: opt.PassportVendorLookup,
	})
	if err != nil {
		err = fmt.Errorf("building workflow context: %w", err)
		return
	}

	action = QueryFieldResultAction{
		node: model.NodeWithCredential{
			Node: model.Node{
				ID: "mockRunNode",
				EditableNode: model.EditableNode{
					Name:         "mockNode",
					Class:        opt.Spec.Class,
					CredentialID: opt.CredentialID, // using the current editing node credential.
				},
				Type: opt.Spec.Type,
				Data: model.NodeData{
					MetaData:    opt.Spec.GenerateNodeMetaData(),
					InputFields: opt.InputFields,
				},
			},
			CredentialWithParent: opt.Credential,
		},
		workflowContext: workflowContext,
	}

	return
}
