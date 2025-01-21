package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNodeOutputVariableReferenceExpression(t *testing.T) {
	type args struct {
		expression string
	}
	tests := []struct {
		name        string
		args        args
		wantNodeID  string
		wantKeyPath string
		wantOk      bool
	}{
		{
			name: "test empty",
			args: args{
				expression: "",
			},
			wantOk: false,
		},
		{
			name: "test error expression",
			args: args{
				expression: ".node.output",
			},
			wantOk: false,
		},
		{
			name: "test {{ .Node.node1.output }}",
			args: args{
				expression: "{{ .Node.node1.output }}",
			},
			wantNodeID: "node1",
			wantOk:     true,
		},
		{
			name: "test {{ .Node.node1.output.a.b.c }}",
			args: args{
				expression: "{{ .Node.node1.output.a.b.c }}",
			},
			wantNodeID:  "node1",
			wantKeyPath: "a.b.c",
			wantOk:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodeID, gotKeyPath, gotOk := ParseNodeOutputVariableReferenceExpression(tt.args.expression)
			assert.Equalf(t, tt.wantNodeID, gotNodeID, "ParseNodeOutputVariableReferenceExpression(%v)", tt.args.expression)
			assert.Equalf(t, tt.wantKeyPath, gotKeyPath, "ParseNodeOutputVariableReferenceExpression(%v)", tt.args.expression)
			assert.Equalf(t, tt.wantOk, gotOk, "ParseNodeOutputVariableReferenceExpression(%v)", tt.args.expression)
		})
	}
}
