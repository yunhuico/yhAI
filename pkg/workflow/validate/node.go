package validate

import (
	"fmt"

	"github.com/spf13/cast"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

var (
	defaultValidateFuncs = []func(model.Node) error{
		validateNodeID,
		validateNodeName,
		validateNodeType,
		validateNodeClass,
		validateNodeData,
		validateNodeInputFields,
	}
)

type validateOpt struct {
	validateNodePropertyFuncs []func(model.Node) error
}
type OptFunc func(opt *validateOpt)

func getValidateOpt(fns []OptFunc) *validateOpt {
	opt := &validateOpt{defaultValidateFuncs}
	for _, f := range fns {
		f(opt)
	}

	return opt
}

// WithImportWorkflowOpt is the validate option used when importing a workflow
// in this situation, we do not check the InputFields of the node.
func WithImportWorkflowOpt() OptFunc {
	return func(opt *validateOpt) {
		opt.validateNodePropertyFuncs = []func(node model.Node) error{
			validateNodeID,
			validateNodeName,
			validateNodeType,
			validateNodeClass,
			validateNodeData,
		}
	}
}

// validateNode validates the node property.
// By default, we check ID, name, type, class, data, inputFields
func validateNode(node model.Node, opt *validateOpt) (err error) {
	for _, f := range opt.validateNodePropertyFuncs {
		if err = f(node); err != nil {
			return
		}
	}
	return
}

func validateNodeClass(node model.Node) error {
	if node.Class == "" {
		return &FieldError{
			field:  "node.class",
			actual: node.Class,
			err:    fmt.Errorf("node class must not be empty"),
		}
	}
	return nil
}

func validateNodeID(node model.Node) error {
	if node.ID != "" {
		if !isAlphaNumeric(node.ID) {
			return &FieldError{
				field:  "node.id",
				actual: node.ID,
				err:    fmt.Errorf("id only alphanumeric characters and '_' are allowed"),
			}
		}
	}
	return nil
}

func validateNodeName(node model.Node) error {
	if node.Name == "" {
		return &FieldError{
			field: "node.name",
			err:   fmt.Errorf("node name must not be empty"),
		}
	}
	return nil
}

func validateNodeType(node model.Node) error {
	switch node.Type {
	case model.NodeTypeLogic, model.NodeTypeActor, model.NodeTypeTrigger:
		return nil
	}

	return &FieldError{
		field:  "node.type",
		actual: string(node.Type),
		err:    fmt.Errorf("node type is not supported"),
	}
}

// validateNodeData validates the node data.
// The data meta data must be valid.
func validateNodeData(node model.Node) error {
	if node.Data.MetaData.AdapterClass == "" {
		return &FieldError{
			field: "node.data.metaData.adapterClass",
			err:   fmt.Errorf("node data adapterClass must not be empty"),
		}
	}

	return nil
}

func validateNodeInputFields(node model.Node) error {
	switch node.Type {
	case model.NodeTypeActor, model.NodeTypeTrigger:
		return nil
	case model.NodeTypeLogic:
		switch node.Class {
		case SwitchClass:
			_, ok1 := node.Data.InputFields["paths"]
			if !ok1 {
				return &FieldError{
					field: "node.data.inputFields",
					err:   fmt.Errorf("switch should have at least one path"),
				}
			}
		case ForeachClass:
			inputCollection, ok := node.Data.InputFields["inputCollection"]
			if !ok || cast.ToString(inputCollection) == "" {
				return &FieldError{
					field: "node.data.inputFields.inputCollection",
					err:   fmt.Errorf("foreach should have inputCollection"),
				}
			}
		default:
			return &FieldError{
				field: "node.Class",
				err:   fmt.Errorf("unrecognized Class name: %v", node.Class),
			}
		}
	default:
		return &FieldError{
			field: "node.Type",
			err:   fmt.Errorf("unrecognized Class Type: %v", node.Type),
		}
	}
	return nil
}
