package share

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type dummyAuthorStore struct{}

func (d dummyAuthorStore) GetUserByID(ctx context.Context, id int) (user model.User, err error) {
	user = model.User{
		ID:   id,
		Name: "Answer to Everything",
	}

	return
}

func (d dummyAuthorStore) GetOrganizationByID(ctx context.Context, id int) (organization model.Organization, err error) {
	organization = model.Organization{
		ID: id,
		EditableOrganization: model.EditableOrganization{
			Name: "Answer to Everything",
		},
	}

	return
}

func TestShareFile(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test file not Exists",
			args: args{
				file: "test.yaml",
			},
			wantErr: true,
		},
		{
			name: "dingtalk test",
			args: args{
				file: "../../../examples/dingtalk-message/dingtalk-message.yaml",
			},
			wantErr: false,
		},
		{
			name: "issue handler test",
			args: args{
				file: "../../../examples/issue-handler/issue-handler.yaml",
			},
			wantErr: false,
		},
		{
			name: "issue-to-pagerduty test",
			args: args{
				file: "../../../examples/issue-to-pagerduty/issue-to-pagerduty.yaml",
			},
		},
		{
			name: "link-issue test",
			args: args{
				file: "../../../examples/link-issue/cs-link-issue.yaml",
			},
		},
		{
			name: "move-issue test",
			args: args{
				file: "../../../examples/move-issue/cs-move-issue.yaml",
			},
		},
		{
			name: "confirm node test",
			args: args{
				file: "../../../examples/cvm/clear-cvm.yaml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testWorkflow, err := workflow.UnmarshalWorkflowFromFile(tt.args.file)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ExportFile() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			testWorkflow.StartNodeID = testWorkflow.Nodes[0].ID

			workflowYamlFromExport, err := Export(context.TODO(), &testWorkflow, dummyAuthorStore{})
			if err != nil && !tt.wantErr {
				t.Errorf("ExportFile() error = %v", err)
				return
			}

			var workflowWithNodes model.WorkflowWithNodes
			yamlBytesFromExport := []byte(workflowYamlFromExport)
			err = yaml.Unmarshal(yamlBytesFromExport, &workflowWithNodes)
			if err != nil {
				t.Errorf("unmarshaling workflow yaml: %v", err)
				return
			}

			err = SanitizeImport(&workflowWithNodes)
			if err != nil && !tt.wantErr {
				t.Errorf("ExportFile() error = %v", err)
				return
			}

			assert.NotEmpty(t, workflowWithNodes.ID)
			assert.NotEqual(t, testWorkflow.StartNodeID, workflowWithNodes.StartNodeID)
			assert.Equal(t, workflowWithNodes.CreatedAt.Truncate(1*time.Hour), time.Now().Truncate(time.Hour))
			assert.Equal(t, workflowWithNodes.UpdatedAt.Truncate(1*time.Hour), time.Now().Truncate(time.Hour))
			for i, node := range workflowWithNodes.Nodes {
				assert.NotEmpty(t, node.ID)
				assert.Empty(t, node.CredentialID)
				assert.NotEqual(t, testWorkflow.Nodes[i].ID, node.ID)
				if testWorkflow.Nodes[i].Transition != "" {
					assert.NotEqual(t, testWorkflow.Nodes[i].Transition, node.Transition)
				}
				assert.Equal(t, testWorkflow.Nodes[i].Class, node.Class)
				assert.Equal(t, testWorkflow.Nodes[i].Name, node.Name)
				assert.Equal(t, model.NodeTestingDefaultStatus, node.TestingStatus)
			}

			// test import workflow from file
			testWorkflowFromFile, err := workflow.UnmarshalWorkflowFromFile(tt.args.file)
			if err != nil && !tt.wantErr {
				t.Errorf("UnmarshalWorkflowFromFile() error = %v", err)
				return
			}
			err = SanitizeImport(&testWorkflowFromFile)
			if err != nil && !tt.wantErr {
				t.Errorf("SanitizeImport() error = %v", err)
				return
			}
			for _, node := range testWorkflowFromFile.Nodes {
				assert.NotEmpty(t, node.ID)
				assert.Empty(t, node.CredentialID)
			}
		})
	}
}

func Test_replaceNodesExpr1(t *testing.T) {
	mapping := map[string]string{
		"juvcwx4brszoa71j": "ju1",
		"lg81j8uufovccuyc": "lg2",
		"cey0p8l8gs72dv1u": "ce3",
	}

	tests := []struct {
		name         string
		workflowYaml string
		old2NewIDMap map[string]string
		want         string
	}{
		{
			name:         "empty",
			workflowYaml: "",
			old2NewIDMap: nil,
			want:         "",
		},
		{
			name:         "empty2",
			workflowYaml: "",
			old2NewIDMap: mapping,
			want:         "",
		},
		{
			name:         "easy",
			workflowYaml: "{{ .Node.lg81j8uufovccuyc.output.datetime }}",
			old2NewIDMap: mapping,
			want:         "{{ .Node.lg2.output.datetime }}",
		},
		{
			name:         "hard",
			workflowYaml: "<! {{ .Node.juvcwx4brszoa71j.output.url }} >{{ .Node.lg81j8uufovccuyc.output.datetime }}{{ .Node.lg81j8uufovccuyc.output.isWeekday }}",
			old2NewIDMap: mapping,
			want:         "<! {{ .Node.ju1.output.url }} >{{ .Node.lg2.output.datetime }}{{ .Node.lg2.output.isWeekday }}",
		},
		{
			name:         "harder",
			workflowYaml: "{{ .Node.juvcwx4brszoa71j.output.labels.[].id }}",
			old2NewIDMap: mapping,
			want:         "{{ .Node.ju1.output.labels.[].id }}",
		},
		{
			name:         "hardest",
			workflowYaml: "{{ .Node.juvcwx4brszoa71j.output.assignees[].emails[]? }}",
			old2NewIDMap: mapping,
			want:         "{{ .Node.ju1.output.assignees[].emails[]? }}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, replaceNodesExpr(tt.workflowYaml, tt.old2NewIDMap), "replaceNodesExpr(%v, %v)", tt.workflowYaml, tt.old2NewIDMap)
		})
	}
}
