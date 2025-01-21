package validate

import (
	"testing"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func TestValidateNodeProperty(t *testing.T) {
	type args struct {
		node model.Node
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test name is empty",
			args: args{
				node: model.Node{
					EditableNode: model.EditableNode{
						Name: "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test id invalid",
			args: args{
				node: model.Node{
					ID: "$invalid",
					EditableNode: model.EditableNode{
						Name: "testWorkflow",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test type invalid",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: "invalid",
					EditableNode: model.EditableNode{
						Name: "testWorkflow",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test node no class",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: model.NodeTypeTrigger,
					EditableNode: model.EditableNode{
						Name: "testWorkflow",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test node no data",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: model.NodeTypeTrigger,
					EditableNode: model.EditableNode{
						Name:  "testWorkflow",
						Class: "testClass",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test node meta data has no app name",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: model.NodeTypeTrigger,
					EditableNode: model.EditableNode{
						Name:  "testWorkflow",
						Class: "testClass",
					},
					Data: model.NodeData{
						MetaData: model.NodeMetaData{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test foreach node no inputFields transition, will pass!",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: model.NodeTypeLogic,
					EditableNode: model.EditableNode{
						Name:  "foreach node",
						Class: ForeachClass,
					},
					Data: model.NodeData{
						MetaData: model.NodeMetaData{
							AdapterClass: "ultrafox/logic",
						},
						InputFields: map[string]any{
							"inputCollection": ".Node.output",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test foreach node no inputFields inputCollection",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: model.NodeTypeLogic,
					EditableNode: model.EditableNode{
						Name:  "foreach node",
						Class: ForeachClass,
					},
					Data: model.NodeData{
						MetaData: model.NodeMetaData{
							AdapterClass: "ultrafox/logic",
						},
						InputFields: map[string]any{
							"transition": "nextNode",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test switch node no inputFields conditions and defaultTransition",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: model.NodeTypeLogic,
					EditableNode: model.EditableNode{
						Name:  "switch node",
						Class: SwitchClass,
					},
					Data: model.NodeData{
						MetaData: model.NodeMetaData{
							AdapterClass: "ultrafox/logic",
						},
						InputFields: map[string]any{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test node valid",
			args: args{
				node: model.Node{
					ID:   "nodeID",
					Type: model.NodeTypeTrigger,
					EditableNode: model.EditableNode{
						Name:  "testWorkflow",
						Class: "testClass",
					},
					Data: model.NodeData{
						MetaData: model.NodeMetaData{
							AdapterClass: "ultrafox/gitlab",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := getValidateOpt(nil)
			if err := validateNode(tt.args.node, opt); (err != nil) != tt.wantErr {
				t.Errorf("validateNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
