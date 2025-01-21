package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_trimBraceBrackets(t *testing.T) {
	type args struct {
		expression string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		matches bool
	}{
		{
			"empty expression",
			args{
				expression: "",
			},
			"",
			false,
		},
		{
			"{{ .Node.node1.output }}",
			args{
				expression: "{{    .Node.node1.output }}",
			},
			".Node.node1.output",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expression, matches := trimBraceBrackets(tt.args.expression)
			assert.Equal(t, tt.want, expression)
			assert.Equal(t, tt.matches, matches)
		})
	}
}
