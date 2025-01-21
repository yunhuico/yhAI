package smtp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_splitLineWriter_Write(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "short",
			input: "12",
			want:  "12",
		},
		{
			name:  "exact",
			input: "1234",
			want:  "1234",
		},
		{
			name:  "more than one lines",
			input: "12345",
			want:  "1234\n5",
		},
		{
			name:  "less than two lines",
			input: "1234567",
			want:  "1234\n567",
		},
		{
			name:  "exact two lines",
			input: "12345678",
			want:  "1234\n5678",
		},
		{
			name:  "more than two lines",
			input: "123456789",
			want:  "1234\n5678\n9",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				assert = require.New(t)
				buf    bytes.Buffer
			)

			l := newSplitLineWriter(&buf, 4, []byte("\n"))
			gotN, err := l.Write([]byte(tt.input))
			assert.NoError(err)
			if gotN != len(tt.input) {
				t.Errorf("Write() gotN = %v, want %v", gotN, len(tt.input))
			}
			output := buf.String()
			assert.Equal(tt.want, output)
		})
	}
}
