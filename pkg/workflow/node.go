package workflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
)

// Kind the kind of node
type Kind string

const (
	TriggerWebhookKind Kind = "webhookTrigger"
	TriggerCronKind    Kind = "cronTrigger"
)

func (k Kind) Type() adapter.AdapterSpecType {
	switch k {
	case TriggerWebhookKind:
		return adapter.SpecTriggerType
	case TriggerCronKind:
		return adapter.SpecTriggerType
	default:
		return adapter.AdapterSpecType(k)
	}
}

func (k Kind) TriggerType() model.TriggerType {
	switch k {
	case TriggerWebhookKind:
		return model.TriggerTypeWebhook
	case TriggerCronKind:
		return model.TriggerTypeCron
	default:
		return ""
	}
}

type Node interface {
	UltrafoxNode() NodeMeta
	Run(c *NodeContext) (output any, err error)
}

type Provisioner interface {
	Provision(ctx context.Context, dependencies ProvisionDeps) (err error)
}

type ProvisionDeps struct {
	Authorizer           auth.Authorizer
	KVProcessor          KVProcessor
	PassportVendorLookup map[model.PassportVendorName]model.PassportVendor
}

func (w *WorkflowContext) newNodeContext(node *nodeInstance) *NodeContext {
	return &NodeContext{
		TriggerNodeContext: NewTriggerNodeContext(w.context),
		node:               node,
		workflowContext:    w,
		serverHost:         w.serverHost,
	}
}

type TriggerNodeContext struct {
	log.Logger

	context context.Context
}

type NodeContext struct {
	TriggerNodeContext

	node            *nodeInstance
	workflowContext *WorkflowContext
	serverHost      *serverhost.ServerHost
}

func (c *NodeContext) Context() context.Context {
	return c.context
}

// NewTriggerNodeContext trigger node just only depends on logger and context for transform data.
func NewTriggerNodeContext(ctx context.Context) TriggerNodeContext {
	return TriggerNodeContext{
		Logger:  log.Clone(log.Namespace("workflow/node")),
		context: ctx,
	}
}

// GetAuthorizer return the authorizer of the node
func (c *NodeContext) GetAuthorizer() auth.Authorizer {
	return c.node.authorizer
}

func (c *NodeContext) evaluate(expr string) (any, error) {
	return c.workflowContext.evaluate(expr)
}
func (c *NodeContext) contextAborted() error {
	return c.workflowContext.contextAborted()
}

// setNextNode input the id of the next node to be executed.
//
// To stop workflow execution, input an empty string.
func (c *NodeContext) setNextNode(transition string) {
	c.workflowContext.nextNodeName = transition
}

func (c *NodeContext) requestLogicControl() {
	c.workflowContext.borrowControl()
}

func (c *NodeContext) clearIter() {
	c.workflowContext.clearScopeIter()
}

func (c *NodeContext) renderTemplate(expr string) ([]byte, error) {
	return c.workflowContext.RenderTemplate(expr)
}

// regOnlyVariableReference just contains one variable reference.
// Y {{ .Node.node1 }}
// Y {{ .Node.node1.output | len }}
// N {{}}
// N {{ }}
// N {{  }}
var regOnlyVariableReference = regexp.MustCompile(`^\{\{\s+([^\{\}|]+)\s+\}\}$`)

// dynamicCalc choose a way to calc given expr, if expr test the regexp pass, use evaluate because
// the expr is a variable reference totally. otherwise, use renderTemplate, because this expr contains
// character entered by the user. transform renderTemplate []byte to string in this function.
func (c *NodeContext) dynamicCalc(expr string) (any, error) {
	if reference, ok := trimBraceBrackets(expr); ok {
		if strings.HasPrefix(reference, ".") { // the pure reference must start with "."
			result, err := c.evaluate(reference)
			if err != nil {
				return nil, fmt.Errorf("dynamic calc reference: %w", err)
			}
			return result, nil
		}
	}

	text, err := c.renderTemplate(expr)
	if err != nil {
		return nil, fmt.Errorf("dynamic calc expr: %w", err)
	}
	return string(text), nil
}

func (c *NodeContext) lookupNode(transition string) *nodeInstance {
	return c.workflowContext.nodeLookup[transition]
}

func (c *NodeContext) unmarshalNode(node *nodeInstance) error {
	return c.workflowContext.unmarshalNodeFromContext(node)
}

func (c *NodeContext) getScopeNodeIDList() []string {
	data := c.workflowContext.scope.getData()
	nodes := make([]string, len(data))
	for node := range data {
		nodes = append(nodes, node)
	}
	return nodes
}

type NodeMeta struct {
	Class     string                  `json:"class"`
	New       func() Node             `json:"-"`
	InputForm adapter.InputFormFields `json:"-"`
}

type nodeInstance struct {
	Node

	ID          string
	Class       string
	Transition  string
	InputFields map[string]any
	authorizer  auth.Authorizer
	inputSchema adapter.InputFormFields

	executed   bool
	startTime  time.Time
	input      any
	output     any
	success    bool
	durationMs int64
}

func NewNodeInstance(ctx context.Context, db *model.DB, cache *cache.Cache, cipher crypto.CryptoCipher, rawNode model.NodeWithCredential) (instance *nodeInstance, err error) {
	if rawNode.ID == "" {
		err = errors.New("node ID can not be empty")
		return
	}

	nodeClass := rawNode.Class
	meta, ok := GetNodeMeta(nodeClass)
	if !ok {
		err = fmt.Errorf("unknown node class %q", rawNode.Class)
		return
	}

	instance = &nodeInstance{
		Node:        meta.New(),
		ID:          rawNode.ID,
		Class:       rawNode.Class,
		Transition:  rawNode.Transition,
		InputFields: rawNode.Data.InputFields,
		inputSchema: getInputSchema(nodeClass),
	}

	if rawNode.CredentialID != "" {
		credential := rawNode.CredentialWithParent.Credential
		if strings.HasPrefix(nodeClass, "ultrafox/slack") {
			if rawNode.Data.InputFields != nil {
				if bot, ok := rawNode.Data.InputFields["bot"]; ok {
					if bot == "" || bot == "yes" {
						credential = rawNode.CredentialWithParent.Parent
					}
				}
			}
		}
		if credential == nil {
			err = fmt.Errorf("node %q related credential %q not exists", rawNode.ID, rawNode.CredentialID)
			return nil, err
		}

		authorizer, err := auth.NewAuthorizer(cipher, credential,
			auth.WithUpdateCredentialTokenFunc(db.UpdateCredentialTokenAndConfirmStatusByID),
			auth.WithOAuthCredentialUpdater(auth.OAuthCredentialUpdater{DB: db.Operator, Cache: cache}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create authorizer for node %q: %w", rawNode.ID, err)
		}
		instance.authorizer = authorizer
	}

	return
}

type QueryFieldResult struct {
	Items      []QueryFieldItem `json:"items"`
	NextCursor string           `json:"nextCursor"`

	// NoMore is true when load all field items.
	// But can't trust this field (in some special api), because some application doesn't provide this flag.
	// You should try you best to calc this field.
	// So, if NoMore is true, stop load. If Items empty, stop load.
	NoMore bool `json:"noMore"`
}

// QueryFieldItem gives a label and a value for the frontend.
//
// User will see the label field, but will submit the value in fact.
type QueryFieldItem struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

// QueryFieldResultProvider
//
// if adapter actor want provider query api that returns a standard resource results,
// should implement this interface.
type QueryFieldResultProvider interface {
	// QueryFieldResultList query the field result list.
	QueryFieldResultList(c *NodeContext) (result QueryFieldResult, err error)
}

// ListPagination for every QueryFieldResultProvider.
type ListPagination struct {
	Page    int    `json:"page"`
	PerPage int    `json:"perPage"`
	Search  string `json:"search"`
	Cursor  string `json:"cursor"`
}

// PreFilterProvider if this trigger request data from callback.
type PreFilterProvider interface {
	GetConfigObject() any

	// PreFilter indicates the request will be ignored if shouldAbort.
	// Webhook service will return error if err != nil
	PreFilter(configObj any, data []byte) (shouldAbort bool, err error)
}
