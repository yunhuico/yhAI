package adapter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_basicValidator_validate(t *testing.T) {
	type args struct {
		inputFields map[string]any
	}
	gitlabTriggerFields := InputFormFields{
		{
			BaseField: BaseField{
				Key:   "scope",
				Label: "Scope",
				Type:  StringFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
			},
		},
		{
			BaseField: BaseField{
				Key:   "projectId",
				Label: "Project",
				Type:  StringFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
				UI: &uiConfig{
					Display: &Display{
						{
							{
								Key:       "scope",
								Operation: displayConditionEquals,
								Value:     "project",
							},
						},
					},
				},
			},
		},
		{
			BaseField: BaseField{
				Key:   "groupId",
				Label: "Group",
				Type:  StringFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
				UI: &uiConfig{
					Display: &Display{
						{
							{
								Key:       "scope",
								Operation: displayConditionEquals,
								Value:     "group",
							},
						},
					},
				},
			},
		},
	}
	queryInTableFields := InputFormFields{
		{
			BaseField: BaseField{
				Key:   "query",
				Label: "Query",
				Type:  ListFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
				Child: &InputFormField{
					BaseField: BaseField{
						Type: StructFieldType,
					},
					AdvancedField: AdvancedField{
						Fields: InputFormFields{
							{
								BaseField: BaseField{
									Key:   "key",
									Label: "Key",
									Type:  StringFieldType,
								},
								AdvancedField: AdvancedField{
									Required: true,
								},
							},
							{
								BaseField: BaseField{
									Key:   "value",
									Label: "Value",
									Type:  StringFieldType,
								},
								AdvancedField: AdvancedField{
									Required: false,
								},
							},
						},
					},
				},
			},
		},
	}
	dingtalkFields := InputFormFields{
		{
			BaseField: BaseField{
				Key:   "type",
				Label: "Type",
				Type:  StringFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
			},
		},
		{
			BaseField: BaseField{
				Key:   "actions",
				Label: "Actions",
				Type:  ListFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
				Child: &InputFormField{
					BaseField: BaseField{
						Type: StringFieldType,
					},
					AdvancedField: AdvancedField{
						Required: true,
					},
				},
				UI: &uiConfig{
					Display: &Display{
						{
							{
								Key:       "type",
								Operation: displayConditionIn,
								Value: []string{
									"textActions",
									"markdown",
								},
							},
						},
					},
				},
			},
		},
	}
	switchFields := InputFormFields{
		{
			BaseField: BaseField{
				Key:   "paths",
				Label: "paths",
				Type:  ListFieldType,
			},
			AdvancedField: AdvancedField{
				Child: &InputFormField{
					BaseField: BaseField{
						Type: StructFieldType,
					},
					AdvancedField: AdvancedField{
						Fields: InputFormFields{
							{
								BaseField: BaseField{
									Key:   "name",
									Label: "Name",
									Type:  StringFieldType,
								},
								AdvancedField: AdvancedField{
									Required: true,
								},
							},
							{
								BaseField: BaseField{
									Key:   "isDefault",
									Label: "Is default?",
									Type:  BoolFieldType,
								},
							},
							{
								BaseField: BaseField{
									Key:   "conditions",
									Label: "Conditions",
									Type:  ListFieldType,
								},
								AdvancedField: AdvancedField{
									Required: false,
									Child: &InputFormField{
										BaseField: BaseField{
											Type: StructFieldType,
										},
										AdvancedField: AdvancedField{
											Required: true,
											Fields: InputFormFields{
												{
													BaseField: BaseField{
														Key:   "left",
														Label: "Left",
														Type:  StringFieldType,
													},
													AdvancedField: AdvancedField{
														Required: true,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	tests := []struct {
		name    string
		fields  InputFormFields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "empty input schema",
			fields: InputFormFields{},
			args:   args{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name: "a not-supported display operation will be ignored, pass anyway",
			fields: InputFormFields{
				{
					BaseField: BaseField{
						Key:   "key",
						Label: "Key",
						Type:  StringFieldType,
					},
					AdvancedField: AdvancedField{
						Required: true,
						UI: &uiConfig{
							Display: &Display{
								{
									{
										Key:       "key",
										Operation: "not_supported",
									},
								},
							},
						},
					},
				},
			},
			args: args{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name: "required field not provided",
			fields: InputFormFields{
				{
					BaseField: BaseField{
						Key:   "projectId",
						Label: "Project",
						Type:  IntFieldType,
					},
					AdvancedField: AdvancedField{
						Required: true,
					},
				},
			},
			args: args{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name: "a not required field, inputFields can be everything",
			fields: InputFormFields{
				{
					BaseField: BaseField{
						Key:   "projectId",
						Label: "Project",
						Type:  IntFieldType,
					},
					AdvancedField: AdvancedField{
						Required: false,
					},
				},
			},
			args: args{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "schema empty, input queryInTableFields not empty, pass",
			fields: InputFormFields{},
			args: args{
				inputFields: map[string]any{
					"k1": "v1",
					"k2": "v2",
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "gitlab conditional required queryInTableFields provided",
			fields: gitlabTriggerFields,
			args: args{
				inputFields: map[string]any{
					"scope":     "project",
					"projectId": "this is project id",
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "gitlab conditional required queryInTableFields don't provide",
			fields: gitlabTriggerFields,
			args: args{
				inputFields: map[string]any{
					"scope":     "group",
					"projectId": "this is project id",
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name:   "gitlab conditional required queryInTableFields don't provide",
			fields: gitlabTriggerFields,
			args: args{
				inputFields: map[string]any{
					"scope":     "group",
					"projectId": "this is project id",
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name: "a required list field's item is not required, but exists empty items, can pass!",
			fields: InputFormFields{
				{
					BaseField: BaseField{
						Key:   "labels",
						Label: "Labels",
						Type:  ListFieldType,
					},
					AdvancedField: AdvancedField{
						Required: true,
						Child: &InputFormField{
							BaseField: BaseField{
								Type: StringFieldType,
							},
							AdvancedField: AdvancedField{
								Required: false,
							},
						},
					},
				},
			},
			args: args{
				inputFields: map[string]any{
					"labels": []string{
						"",
						"prod",
						"",
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name: "a required list field's item is required, but exists a empty item",
			fields: InputFormFields{
				{
					BaseField: BaseField{
						Key:   "labels",
						Label: "Labels",
						Type:  ListFieldType,
					},
					AdvancedField: AdvancedField{
						Required: true,
						Child: &InputFormField{
							BaseField: BaseField{
								Type: StringFieldType,
							},
							AdvancedField: AdvancedField{
								Required: true,
							},
						},
					},
				},
			},
			args: args{
				inputFields: map[string]any{
					"labels": []string{
						"prod",
						"",
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name: "a not required list field's item is required, but exists a empty item",
			fields: InputFormFields{
				{
					BaseField: BaseField{
						Key:   "labels",
						Label: "Labels",
						Type:  ListFieldType,
					},
					AdvancedField: AdvancedField{
						Required: false,
						Child: &InputFormField{
							BaseField: BaseField{
								Type: StringFieldType,
							},
							AdvancedField: AdvancedField{
								Required: true,
							},
						},
					},
				},
			},
			args: args{
				inputFields: map[string]any{
					"labels": []string{
						"prod",
						"",
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name:   "[struct] field have on required field, one not-required field, pass!",
			fields: queryInTableFields,
			args: args{
				inputFields: map[string]any{
					"query": []any{
						map[string]any{
							"key":   "k1",
							"value": "v1",
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "[struct] field have on required field, one not-required field, don't provide query!",
			fields: queryInTableFields,
			args: args{
				inputFields: map[string]any{
					// "query": []any{
					// 	map[string]any{
					// 		"key":   "k1",
					// 		"value": "v1",
					// 	},
					// },
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name:   "[struct] field have on required field, one not-required field, provide a empty-value item, pass!",
			fields: queryInTableFields,
			args: args{
				inputFields: map[string]any{
					"query": []any{
						map[string]any{
							"key":   "k1",
							"value": "",
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "[struct] field have on required field, one not-required field, provide a empty-key item, fail!",
			fields: queryInTableFields,
			args: args{
				inputFields: map[string]any{
					"query": []any{
						map[string]any{
							"key":   "",
							"value": "value",
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name:   "actions is required when type == 'markdown', provide actions, pass!",
			fields: dingtalkFields,
			args: args{
				inputFields: map[string]any{
					"type": "markdown",
					"actions": []string{
						"action 1",
						"action 2",
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "actions is required when type == 'markdown', don't provide actions, fail!",
			fields: dingtalkFields,
			args: args{
				inputFields: map[string]any{
					"type": "markdown",
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name:   "actions is required when type == 'markdown', provide empty actions, fail!",
			fields: dingtalkFields,
			args: args{
				inputFields: map[string]any{
					"type":   "markdown",
					"action": []string{},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name:   "actions is not required when type == 'text', pass!",
			fields: dingtalkFields,
			args: args{
				inputFields: map[string]any{
					"type":   "text",
					"action": []string{},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "switch schema, conditions are empty, pass!",
			fields: switchFields,
			args: args{
				inputFields: map[string]any{
					"paths": []any{
						map[string]any{
							"name":      "branch",
							"isDefault": true,
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name:   "switch schema, one condition left is empty, fail!",
			fields: switchFields,
			args: args{
				inputFields: map[string]any{
					"paths": []any{
						map[string]any{
							"name": "branch",
							"conditions": map[string]any{
								"left": "",
							},
							"isDefault": true,
						},
					},
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := buildBasicValidator(tt.fields)
			tt.wantErr(t, v.validate(tt.args.inputFields), fmt.Sprintf("validate(%v)", tt.args.inputFields))
		})
	}
}

// fixed: https://jihulab.com/ultrafox/ultrafox/-/issues/758
func Test_NumericRequiredField(t *testing.T) {
	v := buildBasicValidator(InputFormFields{
		{
			BaseField: BaseField{
				Key:  "number",
				Type: IntFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
			},
		},
	})
	assert.Error(t, v.validate(map[string]any{}))
	assert.NoError(t, v.validate(map[string]any{"number": 0}))

	v = buildBasicValidator(InputFormFields{
		{
			BaseField: BaseField{
				Key:  "string",
				Type: StringFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
			},
		},
	})
	assert.Error(t, v.validate(map[string]any{"string": ""}))
	assert.NoError(t, v.validate(map[string]any{"string": "UltraFox"}))

	v = buildBasicValidator(InputFormFields{
		{
			BaseField: BaseField{
				Key:  "bool",
				Type: BoolFieldType,
			},
			AdvancedField: AdvancedField{
				Required: true,
			},
		},
	})
	assert.Error(t, v.validate(map[string]any{}))
	assert.NoError(t, v.validate(map[string]any{"bool": true}))
	assert.NoError(t, v.validate(map[string]any{"bool": false}))
}
