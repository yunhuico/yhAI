package payload

import (
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/schema"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/validator"
)

type EditNodeReq struct {
	model.EditableNode

	PreviousNodeInfo

	IsStart     bool           `json:"isStart"`
	InputFields map[string]any `json:"inputFields"`
	ExtFields   map[string]any `json:"extFields"`
}

type PreviousNodeInfo struct {
	PreviousNodeID          string `json:"previousNodeId"`
	PreviousSwitchPathIndex int    `json:"previousSwitchPathIndex"`
	IsFirstInsideNode       bool   `json:"isFirstInsideNode"`
}

type UpdateNodeTransitionReq struct {
	Transition string `json:"transition" yaml:"transition"`
}

type UpdateSwitchNodePathNameReq struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
}

// Normalize payload to node model.
func (p *EditNodeReq) Normalize() (node *model.Node, err error) {
	class := p.Class
	adapterManager := adapter.GetAdapterManager()
	spec := adapterManager.LookupSpec(class)

	if p.IsStart && spec.IsTrigger() {
		// trans field type in backend, because frontend submit all value as string.
		// but static value we need to the actual type for trigger node,
		// the actor node will trans value type automatically.
		p.InputFields, err = schema.RenderInputFieldsBySchema(spec.InputSchema, p.InputFields, schema.NewOriginValueRender())
		if err != nil {
			err = fmt.Errorf("transform trigger node input fields :%w", err)
			return
		}
	}

	err = spec.ValidateDynamically(p.InputFields)
	if err != nil {
		err = fmt.Errorf("validating node dynamically: %w", err)
		return
	}

	node = &model.Node{
		EditableNode: model.EditableNode{
			Name:         p.Name,
			Description:  p.Description,
			Transition:   p.Transition,
			CredentialID: p.CredentialID,
			Class:        class,
		},
		Type: spec.Type,
		Data: model.NodeData{
			MetaData:    spec.GenerateNodeMetaData(),
			InputFields: p.InputFields,
			ExtFields:   p.ExtFields,
		},
	}
	return
}

// Validate the static fields.
func (p *EditNodeReq) Validate() (err error) {
	if err = validator.Validate(p); err != nil {
		return
	}

	class := p.Class
	adapterManager := adapter.GetAdapterManager()
	spec := adapterManager.LookupSpec(class)
	if spec == nil {
		return fmt.Errorf("unknown spec %q", class)
	}

	return
}

type RunNodeReq struct {
	// ParentNodeID if run node in foreach loop, should specify parent node (foreach node) id.
	ParentNodeID string `json:"parentNodeId"`

	// IterIndex can specify the index of foreach data. (default use 0)
	IterIndex int `json:"iterIndex"`
}

type CreateTestSessionReq struct {
	NodeID string `json:"nodeId"`
}

type DecideConfirmReq struct {
	Decision workflow.ConfirmDecision `json:"decision"`
}
