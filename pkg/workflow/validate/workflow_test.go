package validate

import (
	"fmt"
	"testing"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"github.com/stretchr/testify/assert"
)

func TestValidateWorkflowProperty(t *testing.T) {
	type args struct {
		workflow model.WorkflowWithNodes
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test name is empty",
			args: args{
				workflow: model.WorkflowWithNodes{
					Workflow: model.Workflow{
						Name: "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test id invalid",
			args: args{
				workflow: model.WorkflowWithNodes{
					Workflow: model.Workflow{
						ID:   "$invalid",
						Name: "testWorkflow",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test valid workflow",
			args: args{
				workflow: model.WorkflowWithNodes{
					Workflow: model.Workflow{
						ID:   "uuid",
						Name: "test_workflow",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateWorkflowProperty(tt.args.workflow); (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkflowProperty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWorkflow(t *testing.T) {
	type args struct {
		workflow model.WorkflowWithNodes
	}
	tests := []struct {
		name         string
		args         args
		wantRiskLen  int
		wantFatalLen int
	}{
		{
			"test workflow no name",
			args{
				workflow: model.WorkflowWithNodes{
					Workflow: model.Workflow{
						ID: "uuid",
					},
				},
			},
			0,
			1,
		},
		{
			"test workflow no node",
			args{
				workflow: model.WorkflowWithNodes{
					Workflow: model.Workflow{
						ID:   "uuid",
						Name: "test_workflow",
					},
				},
			},
			1,
			0,
		},
		{
			"test workflow two node exists error",
			args{
				workflow: model.WorkflowWithNodes{
					Workflow: model.Workflow{
						ID:   "uuid",
						Name: "test_workflow",
					},
					Nodes: model.Nodes{
						{
							ID: "uuid-uuid",
						},
						{
							ID: "uuid",
							EditableNode: model.EditableNode{
								Name: "",
							},
						},
					},
				},
			},
			0,
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateWorkflow(tt.args.workflow)
			if tt.wantFatalLen+tt.wantRiskLen > 0 {
				assert.True(t, got.ExistsReport())
			}
			if tt.wantFatalLen > 0 {
				assert.True(t, got.ExistsFatal())
			}
			if len(got.Risk) != tt.wantRiskLen {
				t.Errorf("ValidateWorkflow() actual %d risk, want %v", len(got.Risk), tt.wantRiskLen)
			}
			if len(got.Fatal) != tt.wantFatalLen {
				t.Errorf("ValidateWorkflow() actual %d fatal, want %v", len(got.Fatal), tt.wantFatalLen)
			}
		})
	}
}

func TestSynthesisReportError(t *testing.T) {
	report := &SynthesisReport{
		Risk: []error{
			fmt.Errorf("risk1"),
			fmt.Errorf("risk2"),
		},
		Fatal: []error{
			&FieldError{
				field:  "node.id",
				actual: "xxx-xx",
				err:    fmt.Errorf("id invalid"),
			},
		},
	}

	err := report.Error()
	t.Log(err)
}

func Test_validateNodeDAG(t *testing.T) {
	type args struct {
		report *SynthesisReport
		nodes  model.Nodes
	}
	tests := []struct {
		name         string
		args         args
		wantRiskLen  int
		wantFatalLen int
	}{
		{
			name: "test node id exists duplicate",
			args: args{
				report: &SynthesisReport{},
				nodes: model.Nodes{
					{
						ID:           "uuid",
						Type:         model.NodeTypeActor,
						EditableNode: model.EditableNode{},
					},
					{
						ID:           "uuid",
						Type:         model.NodeTypeActor,
						EditableNode: model.EditableNode{},
					},
				},
			},
			wantRiskLen:  1,
			wantFatalLen: 1,
		},
		{
			name: "check transaction not founc",
			args: args{
				report: &SynthesisReport{},
				nodes: model.Nodes{
					{
						ID:   "node1",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "not_found",
						},
					},
				},
			},
			wantFatalLen: 1,
		},
		{
			name: "all node is actor, exists cycle dependency",
			args: args{
				report: &SynthesisReport{},
				nodes: model.Nodes{
					{
						ID:   "node1",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "node2",
						},
					},
					{
						ID:   "node2",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "node3",
						},
					},
					{
						ID:   "node3",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "node1",
						},
					},
				},
			},
			wantFatalLen: 1,
		},
		{
			name: "test foreach node transaction exist cycle",
			args: args{
				report: &SynthesisReport{},
				nodes: model.Nodes{
					{
						ID:   "node1",
						Type: model.NodeTypeTrigger,
						EditableNode: model.EditableNode{
							Transition: "node2",
						},
					},
					{
						ID:   "node2",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "node3foreach",
						},
					},
					{
						ID:   "node3foreach",
						Type: model.NodeTypeLogic,
						EditableNode: model.EditableNode{
							Class:      ForeachClass,
							Transition: "end",
						},
						Data: model.NodeData{
							InputFields: map[string]any{
								"inputCollection": ".",
								"transition":      "foreachItem1",
							},
						},
					},
					{
						ID:   "foreachItem1",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "foreachItem2",
						},
					},
					{
						ID:   "foreachItem2",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "node2",
						},
					},
					{
						ID:   "end",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "",
						},
					},
				},
			},
			wantRiskLen:  0,
			wantFatalLen: 1,
		},
		{
			name: "test switch node transaction exist cycle",
			args: args{
				report: &SynthesisReport{},
				nodes: model.Nodes{
					{
						ID:   "node1",
						Type: model.NodeTypeTrigger,
						EditableNode: model.EditableNode{
							Transition: "node2",
						},
					},
					{
						ID:   "node2",
						Type: model.NodeTypeActor,
						EditableNode: model.EditableNode{
							Transition: "node3switch",
						},
					},
					{
						ID:   "node3switch",
						Type: model.NodeTypeLogic,
						EditableNode: model.EditableNode{
							Class: SwitchClass,
						},
						Data: model.NodeData{
							InputFields: map[string]any{
								"paths": []any{
									map[string]any{
										"name": "path1",
										"conditions": []any{
											[]any{
												map[string]any{
													"left":      "foo",
													"right":     "bar",
													"operation": EqualsOperation,
												},
											},
										},
										"transition": "switchItem1",
									},
									map[string]any{
										"name":       "path2-default",
										"transition": "node2",
										"isDefault":  true,
									},
								},
							},
						},
					},
					{
						ID:           "switchItem1",
						Type:         model.NodeTypeActor,
						EditableNode: model.EditableNode{},
					},
				},
			},
			wantRiskLen:  0,
			wantFatalLen: 1,
		},
		{
			"test foreach node data inputFields invalid",
			args{
				report: &SynthesisReport{},
				nodes: model.Nodes{
					{
						ID:   "foreach",
						Type: model.NodeTypeLogic,
						EditableNode: model.EditableNode{
							Class: ForeachClass,
						},
						Data: model.NodeData{
							InputFields: map[string]any{
								"inputCollection": 9999,
								"transition":      "foreachItem1",
							},
						},
					},
				},
			},
			0,
			1,
		},
		{
			"test switch node data inputFields invalid, transition to a not defined node",
			args{
				report: &SynthesisReport{},
				nodes: model.Nodes{
					{
						ID:   "switch",
						Type: model.NodeTypeLogic,
						EditableNode: model.EditableNode{
							Class: SwitchClass,
						},
						Data: model.NodeData{
							InputFields: map[string]any{
								"paths": []any{
									map[string]any{
										"conditions": []any{
											[]any{
												map[string]any{},
											},
										},
										"transition": "not_defined_node",
									},
								},
							},
						},
					},
				},
			},
			0,
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateNodeDAG(tt.args.report, tt.args.nodes)
			if len(tt.args.report.Risk) != tt.wantRiskLen {
				t.Errorf("ValidateWorkflow() actual %d risk, want %v", len(tt.args.report.Risk), tt.wantRiskLen)
			}
			if len(tt.args.report.Fatal) != tt.wantFatalLen {
				t.Errorf("ValidateWorkflow() actual %d fatal, want %v", len(tt.args.report.Fatal), tt.wantFatalLen)
			}
		})
	}
}
