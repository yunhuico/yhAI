package schema

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
)

type mockRender struct {
	errorValues []string
}

func (m *mockRender) Read(value string) (string, error) {
	for _, key := range m.errorValues {
		if key == value {
			return "", fmt.Errorf("mock error")
		}
	}
	return value, nil
}

func TestSchema_RenderJSON(t *testing.T) {
	schema := []*adapter.InputFormField{
		{
			BaseField: adapter.BaseField{
				Key:  "projectID",
				Type: adapter.IntFieldType,
			},
			AdvancedField: adapter.AdvancedField{
				Required: true,
			},
		},
		{
			BaseField: adapter.BaseField{
				Key:  "issueID",
				Type: adapter.IntFieldType,
			},
			AdvancedField: adapter.AdvancedField{
				Required: true,
			},
		},
		{
			BaseField: adapter.BaseField{
				Key:  "title",
				Type: adapter.StringFieldType,
			},
			AdvancedField: adapter.AdvancedField{
				Required: true,
			},
		},
		{
			BaseField: adapter.BaseField{
				Key:  "body",
				Type: adapter.StringFieldType,
			},
		},
		{
			BaseField: adapter.BaseField{
				Key:  "labels",
				Type: adapter.ListFieldType,
			},
			AdvancedField: adapter.AdvancedField{
				Child: &adapter.InputFormField{
					BaseField: adapter.BaseField{
						Type: adapter.StringFieldType,
					},
				},
				Required: false,
			},
		},
		{
			BaseField: adapter.BaseField{
				Key:  "due_date",
				Type: adapter.StringFieldType,
			},
		},
	}

	t.Run("input fields empty, pass!", func(t *testing.T) {
		_, err := RenderJSON(schema, map[string]any{}, nil)
		assert.NoError(t, err)
	})

	t.Run("input projectID is bool, ignore this field!", func(t *testing.T) {
		_, err := RenderJSON(schema, map[string]any{
			"projectID": true,
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("input projectID is int, test pass!", func(t *testing.T) {
		_, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("input title is int, int trans string", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     12,
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, `{"issueID":1,"projectID":12,"title":"12"}`, string(result))
	})

	t.Run("input title is bool, bool trans string", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     true,
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"issueID":1,"projectID":12,"title":"true"}`), result)
	})

	t.Run("input title is float, float trans string", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     float32(12.12),
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"issueID":1,"projectID":12,"title":"12.12"}`), result)
	})

	t.Run("input title is struct, cannot trans, but ignore error", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     struct{}{},
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"issueID":1,"projectID":12}`), result)
	})

	t.Run("input title is string", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     "this is title",
		}, &mockRender{})
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"issueID":1,"projectID":12,"title":"this is title"}`), result)
	})

	t.Run("input due_date parse error, but ignore error", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     "this is title",
			"due_date":  "okok",
		}, &mockRender{[]string{"okok"}})
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"issueID":1,"projectID":12,"title":"this is title"}`), result)
	})

	t.Run("labels is ok", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     "this is title",
			"labels":    []any{"label1", "label2"},
		}, &mockRender{})
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"issueID":1,"labels":["label1","label2"],"projectID":12,"title":"this is title"}`), result)
	})

	// labels: [label1, label2], will cause error when render label2,
	// so ignore the labels field.
	t.Run("one label is error, so remove the labels", func(t *testing.T) {
		result, err := RenderJSON(schema, map[string]any{
			"projectID": 12,
			"issueID":   1,
			"title":     "this is title",
			"labels":    []any{"label1", "label2"},
		}, &mockRender{[]string{"label2"}})
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"issueID":1,"projectID":12,"title":"this is title"}`), result)
	})

	t.Run("test condition struct", func(t *testing.T) {
		schema = adapter.InputFormFields{
			{
				BaseField: adapter.BaseField{
					Key:  "conditions",
					Type: adapter.ListFieldType,
				},
				AdvancedField: adapter.AdvancedField{
					Child: &adapter.InputFormField{
						BaseField: adapter.BaseField{
							Type: adapter.StructFieldType,
						},
						AdvancedField: adapter.AdvancedField{
							Fields: adapter.InputFormFields{
								{
									BaseField: adapter.BaseField{
										Key:  "expr",
										Type: adapter.StringFieldType,
									},
									AdvancedField: adapter.AdvancedField{
										Required: true,
									},
								},
								{
									BaseField: adapter.BaseField{
										Key:  "transition",
										Type: adapter.StringFieldType,
									},
									AdvancedField: adapter.AdvancedField{
										Required: true,
									},
								},
							},
						},
					},
					Required: false,
				},
			},
		}

		result, err := RenderJSON(schema, map[string]any{
			"conditions": []any{
				map[string]any{
					"expr":       "expr1",
					"transition": "transition1",
				},
				map[string]any{
					"expr":       "expr2",
					"transition": "transition2",
				},
			},
		}, &mockRender{[]string{}})
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"conditions":[{"expr":"expr1","transition":"transition1"},{"expr":"expr2","transition":"transition2"}]}`), result)
	})

	t.Run("array in array", func(t *testing.T) {
		schema = adapter.InputFormFields{
			{
				BaseField: adapter.BaseField{
					Key:  "projects",
					Type: adapter.ListFieldType,
				},
				AdvancedField: adapter.AdvancedField{
					Child: &adapter.InputFormField{
						BaseField: adapter.BaseField{
							Type: adapter.StructFieldType,
						},
						AdvancedField: adapter.AdvancedField{
							Required: false,
							Fields: adapter.InputFormFields{
								{
									BaseField: adapter.BaseField{
										Key:  "issues",
										Type: adapter.ListFieldType,
									},
									AdvancedField: adapter.AdvancedField{
										Child: &adapter.InputFormField{
											BaseField: adapter.BaseField{
												Type: adapter.StructFieldType,
											},
											AdvancedField: adapter.AdvancedField{
												Fields: adapter.InputFormFields{
													{
														BaseField: adapter.BaseField{
															Key:  "labels",
															Type: adapter.ListFieldType,
														},
														AdvancedField: adapter.AdvancedField{
															Child: &adapter.InputFormField{
																BaseField: adapter.BaseField{
																	Type: adapter.StringFieldType,
																},
															},
															Required: false,
														},
													},
												},
											},
										},
										Required: false,
									},
								},
							},
						},
					},
				},
			},
		}

		result, err := RenderJSON(schema, map[string]any{
			"projects": []map[string]any{
				{
					"issues": []map[string]any{
						{
							"labels": []any{
								"label1",
								"label2",
							},
						},
						{
							"labels": []any{
								"label3",
								"label4",
							},
						},
					},
				},
			},
		}, &mockRender{[]string{}})
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"projects":[{"issues":[{"labels":["label1","label2"]},{"labels":["label3","label4"]}]}]}`), result)
	})
}

func Test_parseValue(t *testing.T) {
	type args struct {
		str  string
		kind adapter.FieldType
	}
	tests := []struct {
		name    string
		args    args
		wantV   interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "tran \"40.00\" to 40",
			args: args{
				str:  "40.00",
				kind: adapter.IntFieldType,
			},
			wantV: 40,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name: "tran \"40.00001\" failed",
			args: args{
				str:  "40.00001",
				kind: adapter.IntFieldType,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotV, err := parseValue(tt.args.str, tt.args.kind)
			if !tt.wantErr(t, err, fmt.Sprintf("parseValue(%v, %v)", tt.args.str, tt.args.kind)) {
				return
			}
			assert.Equalf(t, tt.wantV, gotV, "parseValue(%v, %v)", tt.args.str, tt.args.kind)
		})
	}
}
