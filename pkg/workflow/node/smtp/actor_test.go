package smtp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_removeEmptyItem(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "no match",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "one match",
			input: []string{"a", "", "c"},
			want:  []string{"a", "c"},
		},
		{
			name:  "two matches",
			input: []string{"a", "", "c", ""},
			want:  []string{"a", "c"},
		},
		{
			name:  "all blanks",
			input: []string{"", ""},
			want:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotO := removeEmptyItem(tt.input); !reflect.DeepEqual(gotO, tt.want) {
				t.Errorf("removeEmptyItem() = %v, want %v", gotO, tt.want)
			}
		})
	}
}

func TestAttachment_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "empty",
			content: "",
			want:    `{"content":"", "fileName":"empty"}`,
		},
		{
			name:    "short",
			content: "short",
			want:    `{"content":"short", "fileName":"short"}`,
		},
		{
			name:    "long",
			content: "1234567890abc",
			want:    `{"content":"1234567890...(omitted)", "fileName":"long"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)

			a := Attachment{
				FileName: tt.name,
				Content:  tt.content,
			}
			got, err := a.MarshalJSON()
			assert.NoError(err)
			assert.JSONEq(tt.want, string(got))
		})
	}
}
