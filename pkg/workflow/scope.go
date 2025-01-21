package workflow

import (
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/schema"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/template"
)

var _ schema.FieldValueReader = (*contextScopeReader)(nil)

type contextScopeReader struct {
	scope    contextScope
	template *template.Engine
}

func newContextScopeReader(scope contextScope) *contextScopeReader {
	return &contextScopeReader{
		template: template.NewTemplateEngine(scope),
	}
}

func (c contextScopeReader) Read(value string) (string, error) {
	result, err := c.template.RenderTemplate(value)
	if err != nil {
		return "", fmt.Errorf("read value %q error: %v", value, err)
	}
	return string(result), nil
}

type contextScope interface {
	template.ScopeDataProvider
	evaluate(expr string) (any, error)
	// getData returns the scope all data
	getData() map[string]any
	// getDiffNodeData returns the diff node data from the upstream.
	// when confirm and foreach occurs diff data.
	getDiffNodeData() map[string]any
	getNodeData(nodeName string) any
	deleteNode(nodeName string)
	setNodeData(nodeName string, data any)
	setIterData(data map[string]any)
	setIterKeyValue(key string, data any)
	clearIter()
	cloneProxy() contextScope
	setData(data map[string]any)
}

type contextScopeImpl struct {
	Node map[string]any `json:"node,omitempty"`
	Var  map[string]any `json:"var,omitempty"`
	Iter map[string]any `json:"iter,omitempty"`
}

func newContextScope() contextScope {
	return &contextScopeImpl{
		Node: map[string]any{},
		Var:  map[string]any{},
		Iter: map[string]any{},
	}
}

func (c *contextScopeImpl) GetScopeData() map[string]any {
	return c.getData()
}

func (c *contextScopeImpl) evaluate(expr string) (any, error) {
	templateEngine := template.NewTemplateEngine(c)

	return templateEngine.Evaluate(expr)
}

func (c *contextScopeImpl) setData(data map[string]any) {
	node := data["Node"]
	if nodeMap, ok := node.(map[string]any); ok {
		c.Node = nodeMap
	}
	v := data["Var"]
	if varMap, ok := v.(map[string]any); ok {
		c.Var = varMap
	}
	iter := data["Iter"]
	if iterMap, ok := iter.(map[string]any); ok {
		c.Iter = iterMap
	}
}

func (c *contextScopeImpl) getData() map[string]any {
	return map[string]any{
		"Node": c.Node,
		"Var":  c.Var,
		"Iter": c.Iter,
	}
}

func (c *contextScopeImpl) getNodeData(nodeName string) any {
	return c.Node[nodeName]
}

func (c *contextScopeImpl) setNodeData(nodeName string, data any) {
	c.Node[nodeName] = data
}

func (c *contextScopeImpl) setIterKeyValue(key string, data any) {
	c.Iter[key] = data
}

func (c *contextScopeImpl) deleteNode(nodeName string) {
	delete(c.Node, nodeName)
}

func (c *contextScopeImpl) clearIter() {
	c.Iter = map[string]any{}
}

func (c *contextScopeImpl) getDiffNodeData() map[string]any {
	return c.Node
}

func (c *contextScopeImpl) setIterData(data map[string]any) {
	c.Iter = data
}

// contextScopeProxy is a proxy of contextScope, but trace all the calls to upstream.
// we need know which nodes have been added to the scope, and these values need to be fetched separately.
type contextScopeProxy struct {
	diffNodes set.Set[string]
	upstream  contextScope
}

func (c *contextScopeImpl) cloneProxy() contextScope {
	return &contextScopeProxy{
		diffNodes: set.Set[string]{},
		upstream:  c.clone(),
	}
}

func (c *contextScopeProxy) setData(data map[string]any) {
	c.upstream.setData(data)
}

func (c *contextScopeImpl) clone() *contextScopeImpl {
	return &contextScopeImpl{
		Node: cloneMap(c.Node),
		Var:  cloneMap(c.Var),
		Iter: cloneMap(c.Iter),
	}
}

func cloneMap(m map[string]any) map[string]any {
	result := map[string]any{}
	for k, v := range m {
		result[k] = v
	}
	return result
}

func (c *contextScopeProxy) cloneProxy() contextScope {
	return &contextScopeProxy{
		diffNodes: set.Set[string]{},
		upstream:  c,
	}
}

func (c *contextScopeProxy) evaluate(expr string) (any, error) {
	return c.upstream.evaluate(expr)
}

func (c *contextScopeProxy) getData() map[string]any {
	return c.upstream.getData()
}

func (c *contextScopeProxy) setNodeData(nodeName string, data any) {
	c.diffNodes.Add(nodeName)
	c.upstream.setNodeData(nodeName, data)
}

func (c *contextScopeProxy) getDiffNodeData() map[string]any {
	result := map[string]any{}
	// find the root scope
	var root *contextScopeImpl
	cur := c.upstream
	for {
		if v, ok := cur.(*contextScopeImpl); ok {
			root = v
			break
		} else {
			cur = cur.(*contextScopeProxy).upstream
		}
	}
	c.diffNodes.Foreach(func(nodeName string) {
		result[nodeName] = root.Node[nodeName]
	})
	return result
}

func (c *contextScopeProxy) deleteNode(nodeName string) {
	c.upstream.deleteNode(nodeName)
}

func (c *contextScopeProxy) setIterKeyValue(key string, data any) {
	// subflow cannot call this
	c.upstream.setIterKeyValue(key, data)
}

func (c *contextScopeProxy) clearIter() {
	// subflow cannot call this
	c.upstream.clearIter()
}

func (c *contextScopeProxy) getNodeData(nodeName string) any {
	return c.upstream.getNodeData(nodeName)
}

func (c *contextScopeProxy) GetScopeData() map[string]any {
	return c.getData()
}

func (c *contextScopeProxy) setIterData(data map[string]any) {
	c.upstream.setIterData(data)
}
