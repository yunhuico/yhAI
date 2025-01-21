package utils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatJSONIndent(t *testing.T) {
	type args struct {
		msg json.RawMessage
	}
	tests := []struct {
		name           string
		args           args
		wantIndentJSON string
		wantErr        bool
	}{
		{
			name: "test null",
			args: args{
				msg: json.RawMessage(`null`),
			},
			wantIndentJSON: "null",
			wantErr:        false,
		},
		{
			name: "test {}",
			args: args{
				msg: json.RawMessage(`{}`),
			},
			wantIndentJSON: "{}",
			wantErr:        false,
		},
		{
			name: "test string",
			args: args{
				msg: json.RawMessage(`"ultrafox"`),
			},
			wantIndentJSON: "\"ultrafox\"",
			wantErr:        false,
		},
		{
			name: "test array",
			args: args{
				msg: json.RawMessage(`["foo", "bar"]`),
			},
			wantIndentJSON: `[
 "foo",
 "bar"
]`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndentJSON, err := FormatJSONIndent(tt.args.msg)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Equalf(t, tt.wantIndentJSON, gotIndentJSON, "FormatJSONIndent(%v)", tt.args.msg)
		})
	}
}
