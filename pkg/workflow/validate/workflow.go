package validate

import (
	"fmt"
	"io"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/console"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate/dag"
)

// FieldError is a field error
type FieldError struct {
	field  string
	actual string
	err    error
}

func (f *FieldError) Error() string {
	return fmt.Sprintf("field %q error: %s", f.field, f.err)
}

// SynthesisReport try to synthesis all possible error.
// Have two scenes to use the report:
// if in cli, just use the Error() direction.
// if in api, can handle the error by any way.
type SynthesisReport struct {
	Risk  []error
	Fatal []error
}

func (r *SynthesisReport) ExistsReport() bool {
	return len(r.Fatal) > 0 || len(r.Risk) > 0
}

func (r *SynthesisReport) ExistsFatal() bool {
	return len(r.Fatal) > 0
}

func (r *SynthesisReport) LogString() string {
	var errStr string
	for _, err := range r.Fatal {
		errStr += err.Error() + "\n"
	}
	for _, err := range r.Risk {
		errStr += err.Error() + "\n"
	}
	return errStr
}

// nolint
func (r *SynthesisReport) Warning(w io.Writer) {
	if len(r.Risk) > 0 {
		w.Write([]byte(console.RenderGreyColor("⚠️ Some risk you should notice:").String()))
		w.Write([]byte("\n"))
		for i, risk := range r.Risk {
			w.Write([]byte(fmt.Sprintf("#%d ", i+1)))
			w.Write([]byte(risk.Error()))
			w.Write([]byte("\n"))
		}
	}
}

func (r *SynthesisReport) Error() string {
	if len(r.Fatal) == 0 && len(r.Risk) == 0 {
		return "no error"
	}

	strBuf := strings.Builder{}
	r.Warning(&strBuf)
	if len(r.Fatal) > 0 {
		strBuf.WriteString(console.RenderGreyColor("️❌ Some error you should fix:").String())
		strBuf.WriteString("\n")
		for i, fatal := range r.Fatal {
			strBuf.WriteString(fmt.Sprintf("#%d ", i+1))
			strBuf.WriteString(fatal.Error())
			strBuf.WriteString("\n")
		}
	}
	return strBuf.String()
}

// ValidateWorkflow a workflow, return a synthesis report.
// just check the static properties, not contains the dynamic properties like connectionID.
// not check cycle transition.
// not check the dynamic properties like connectionID. the dynamic properties depends on external data.
func ValidateWorkflow(workflow model.WorkflowWithNodes, validateNodeOpts ...OptFunc) *SynthesisReport {
	report := &SynthesisReport{}
	if err := ValidateWorkflowProperty(workflow); err != nil {
		report.Fatal = append(report.Fatal, err)
		return report
	}

	if len(workflow.Nodes) == 0 {
		report.Risk = append(report.Risk, fmt.Errorf("workflow must have at least one node"))
		return report
	}

	// validate node static property
	opt := getValidateOpt(validateNodeOpts)
	for _, node := range workflow.Nodes {
		if err := validateNode(node, opt); err != nil {
			report.Fatal = append(report.Fatal, err)
		}
	}

	if report.ExistsReport() {
		return report
	}

	validateNodeDAG(report, workflow.Nodes)

	return report
}

// TODO(sword): refactor this function more pure.
func validateNodeDAG(report *SynthesisReport, nodes model.Nodes) {
	nodeMap := nodes.MapByID()
	// foreach node is specified, we can think of the loop body in foreach as a separate workflow.
	// but as the same way, also should make sure that foreach node dependency relation with other nodes.
	foreachNodeTransition := map[string]string{}
	dependencyRelation := map[string][]string{}
	dag := dag.NewDag()

	// if check pass, return true.
	checkIDDuplicate := func(id string, nodeName string) bool {
		dupNodeName, ok := dependencyRelation[id]
		if !ok {
			return true
		}
		report.Fatal = append(report.Fatal, fmt.Errorf("node id %q is duplicate, in both %q and %q node", id, dupNodeName, nodeName))
		return false
	}

	// if check pass, return true.
	checkTransitionNodeExists := func(transition string) bool {
		if transition == "" {
			return true
		}
		if _, ok := nodeMap[transition]; !ok {
			report.Fatal = append(report.Fatal, fmt.Errorf("node %q not exists", transition))
			return false
		}
		return true
	}

	// build relation data
	for i, node := range nodes {
		if !checkIDDuplicate(node.ID, node.Name) {
			return
		}
		if !checkTransitionNodeExists(node.Transition) {
			return
		}

		// switch no transition property, because switch will choose a path transition.
		if node.Transition != "" && node.Class != SwitchClass {
			dag.AddEdge(node.ID, node.Transition)
		} else if i != len(nodes)-1 && node.Class != SwitchClass {
			report.Risk = append(report.Risk, fmt.Errorf("node %q does not have any specified transition", node.Name))
		}

		switch node.Type {
		case model.NodeTypeActor:
			dependencyRelation[node.ID] = append(dependencyRelation[node.ID], node.Transition)
		case model.NodeTypeLogic:
			switch node.Class {
			case ForeachClass:
				var foreachNode LoopFromListNode
				err := trans.MapToStruct(node.Data.InputFields, &foreachNode)
				if err != nil {
					report.Fatal = append(report.Fatal, fmt.Errorf("foreach input field error: %s", err))
					return
				}
				if !checkTransitionNodeExists(foreachNode.Transition) {
					return
				}
				dag.AddEdge(node.ID, foreachNode.Transition)
				dependencyRelation[node.ID] = append(dependencyRelation[node.ID], foreachNode.Transition)
				foreachNodeTransition[node.ID] = foreachNode.Transition
			case SwitchClass:
				var switchNode SwitchLogicNode
				err := trans.MapToStruct(node.Data.InputFields, &switchNode)
				if err != nil {
					report.Fatal = append(report.Fatal, fmt.Errorf("switch input field error: %s", err))
					return
				}
				for _, path := range switchNode.Paths {
					if !checkTransitionNodeExists(path.Transition) {
						return
					}
					dag.AddEdge(node.ID, path.Transition)
					dependencyRelation[node.ID] = append(dependencyRelation[node.ID], path.Transition)
				}
			}
		case model.NodeTypeTrigger:
			dependencyRelation[node.ID] = append(dependencyRelation[node.ID], node.Transition)
		}
	}

	// use algorithm to check the dag.
	if err := dag.Build(); err != nil {
		report.Fatal = append(report.Fatal, err)
		return
	}
}

// ValidateWorkflowProperty validates the workflow property
// if id not empty, check it
func ValidateWorkflowProperty(workflow model.WorkflowWithNodes) error {
	if workflow.ID != "" {
		if err := validateWorkflowID(workflow.ID); err != nil {
			return &FieldError{field: "workflow.id", actual: workflow.ID, err: err}
		}
	}

	if workflow.Name == "" {
		return &FieldError{field: "workflow.name", err: fmt.Errorf("workflow name is empty")}
	}

	return nil
}

func validateWorkflowID(id string) error {
	return validateID(id)
}

// validateID id can only contain alphanumeric characters and '_'.
func validateID(id string) error {
	if !isAlphaNumeric(id) {
		return fmt.Errorf("invalid id %q, only alphanumeric characters and '_' are allowed", id)
	}

	return nil
}

func isAlphaNumeric(s string) bool {
	for _, c := range s {
		if !isAlphaNumericChar(c) {
			return false
		}
	}

	return true
}

func isAlphaNumericChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}
