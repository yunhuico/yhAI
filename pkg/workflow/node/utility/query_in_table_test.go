package utility

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryInTable_Run(t *testing.T) {
	type fields struct {
		FieldKey          string
		QueryTable        []TableItem
		CompensationValue string
	}
	tests := []struct {
		name       string
		fields     fields
		wantResult any
	}{
		{
			name: "don't define table",
			fields: fields{
				FieldKey:          "foo",
				QueryTable:        nil,
				CompensationValue: "",
			},
			wantResult: map[string]any{
				"data": "",
			},
		},
		{
			name: "don't define table, return compensation value",
			fields: fields{
				FieldKey:          "foo",
				QueryTable:        nil,
				CompensationValue: "bar",
			},
			wantResult: map[string]any{
				"data": "bar",
			},
		},
		{
			name: "field matched",
			fields: fields{
				FieldKey: "k1",
				QueryTable: []TableItem{
					{
						Key:   "k1",
						Value: "v1",
					},
				},
				CompensationValue: "",
			},
			wantResult: map[string]any{
				"data": "v1",
			},
		},
		{
			name: "not matched, use compensation value",
			fields: fields{
				FieldKey: "k2",
				QueryTable: []TableItem{
					{
						Key:   "k1",
						Value: "v1",
					},
				},
				CompensationValue: "default",
			},
			wantResult: map[string]any{
				"data": "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := QueryInTable{
				FieldKey:     tt.fields.FieldKey,
				QueryTable:   tt.fields.QueryTable,
				DefaultValue: tt.fields.CompensationValue,
			}
			gotResult, err := q.Run(nil)
			assert.NoError(t, err)

			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("Run() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
