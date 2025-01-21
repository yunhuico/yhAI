package dag

import (
	"fmt"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"
)

// Dag is a directed acyclic graph of nodes.
type Dag struct {
	edges   []Edge
	fromMap map[string][]string
	toMap   map[string][]string
	tree    tree
}

// Edge from one node to another, so From and To are node name.
type Edge struct {
	From string
	To   string
}

var (
	// ErrCycle is returned when a cycle is detected in the graph.
	ErrCycle = fmt.Errorf("cycle detected")
)

type node struct {
	name     string
	children tree
}

type tree []*node

func NewDag() *Dag {
	return &Dag{
		edges: []Edge{},
	}
}

func (d *Dag) AddEdge(from, to string) {
	d.edges = append(d.edges, Edge{
		From: from,
		To:   to,
	})
}

type path struct {
	path    []string // [node1, node2, node3]
	nodeSet set.Set[string]
}

func newPath() *path {
	return &path{
		path:    []string{},
		nodeSet: set.Set[string]{},
	}
}

func (p path) printCyclePath(startNode string) string {
	var startIndex int
	for i, node := range p.path {
		if node == startNode {
			startIndex = i
		}
	}
	return strings.Join(p.path[startIndex:], " -> ") + " -> " + startNode
}

func (p path) copy() *path {
	return &path{
		path:    append([]string{}, p.path...),
		nodeSet: p.nodeSet.Copy(),
	}
}

// Build builds the tree. if exists error, then must exist a cycle.
func (d *Dag) Build() error {
	if len(d.edges) == 0 {
		return nil
	}

	var err error
	d.buildRelation()

	roots := d.findRoots()
	if len(roots) == 0 {
		return fmt.Errorf("cannot find any root, because all edge become a cycle")
	}

	tree := make(tree, len(roots))
	for i, root := range roots {
		tree[i], err = d.buildChildren(root, d.fromMap[root], newPath())
		if err != nil {
			return err
		}
	}
	d.tree = tree
	return nil
}

// TODO: optimize the code, cache the success path, but no need for now.
func (d *Dag) buildChildren(rootName string, nodeNames []string, path *path) (*node, error) {
	if len(nodeNames) == 0 { // leaf node
		return &node{
			name: rootName,
		}, nil
	}

	if path.nodeSet.Has(rootName) {
		startNode := rootName // cycle start from this node
		return nil, fmt.Errorf("%s cycle detected: %w", path.printCyclePath(startNode), ErrCycle)
	}

	var err error
	children := make(tree, len(nodeNames))
	path.path = append(path.path, rootName)
	path.nodeSet.Add(rootName)

	for i, childName := range nodeNames {
		children[i], err = d.buildChildren(childName, d.fromMap[childName], path.copy())
		if err != nil {
			return nil, err
		}
	}
	return &node{
		name:     rootName,
		children: children,
	}, nil
}

func (d *Dag) findRoots() []string {
	roots := []string{}
	for _, edge := range d.edges {
		if _, ok := d.toMap[edge.From]; !ok {
			roots = append(roots, edge.From)
		}
	}
	return roots
}

func (d *Dag) buildRelation() {
	toMap := map[string][]string{}
	for _, edge := range d.edges {
		toMap[edge.To] = append(toMap[edge.To], edge.From)
	}
	d.toMap = toMap

	fromMap := map[string][]string{}
	for _, edge := range d.edges {
		fromMap[edge.From] = append(fromMap[edge.From], edge.To)
	}
	d.fromMap = fromMap
}
