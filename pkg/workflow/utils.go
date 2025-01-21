package workflow

import (
	"regexp"
)

// use this regex get nodeID and key path.
// {{ .Node.nodeID.output }}
// {{ .Node.nodeID.output.list }}
// {{ .Node.nodeID.output.issue.labels }}
// {{ .Node.nodeID.output.issue.labels2 }}
// .Node.nodeID.output
// .Node.nodeID.output.list
// .Node.nodeID.output.issue.labels
// .Node.nodeID.output.issue.labels2
// DEPRECATED
var regexNodeOutputVariableReference = regexp.MustCompile(`\.Node\.([\w\d]+)\.output.?([\w\d\.\_]+)?`)

// ParseNodeOutputVariableReferenceExpression
// DEPRECATED: format output will be removed.
func ParseNodeOutputVariableReferenceExpression(expression string) (nodeID, keyPath string, ok bool) {
	subMatches := regexNodeOutputVariableReference.FindStringSubmatch(expression)
	if len(subMatches) != 3 {
		return
	}

	nodeID = subMatches[1]
	keyPath = subMatches[2]
	ok = true
	return
}
