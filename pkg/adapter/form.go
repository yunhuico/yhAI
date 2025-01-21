package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
)

const defaultLanguage = "en-US"

type Lang struct {
	EN   string `json:"en-US"` // nolint
	ZhCN string `json:"zh-CN"` // nolint
}

func (l *Lang) Defined() bool {
	if l == nil {
		return false
	}
	return l.EN != "" || l.ZhCN != ""
}

func (l *Lang) UnmarshalJSON(data []byte) (err error) {
	en, err := unmarshalString(data)
	if err == nil {
		l.EN = en
		return
	}

	m := make(map[string]json.RawMessage)
	if err = json.Unmarshal(data, &m); err != nil {
		err = fmt.Errorf("unmarshal lang: %w", err)
		return
	}
	if err = unmarshalKey("en-US", m, &l.EN); err != nil {
		err = fmt.Errorf("unmarshal lang en: %w", err)
		return
	}
	if err = unmarshalKey("zh-CN", m, &l.ZhCN); err != nil {
		err = fmt.Errorf("unmarshal lang zh-cn: %w", err)
		return
	}

	return
}

// GetLang default is english.
func (l *Lang) GetLang(lang string) (text string) {
	defer func() {
		if text == "" {
			text = l.EN
		}
	}()

	if lang == "zh-CN" {
		text = l.ZhCN
	}

	return
}

type internalInputFormFields []internalInputFormField

// ToDefault returns the default english language version for each value field.
func (fields internalInputFormFields) ToDefault() (result InputFormFields) {
	return fields.ToLang(defaultLanguage)
}

func (fields internalInputFormFields) ToLang(lang string) (result InputFormFields) {
	result = make(InputFormFields, len(fields))
	for i, field := range fields {
		result[i] = field.ToLang(lang)
	}
	return
}

func (field internalInputFormField) ToLang(lang string) (result *InputFormField) {
	result = &InputFormField{
		BaseField: BaseField{
			Key:   field.Key,
			Label: field.Label.GetLang(lang),
			Type:  field.Type,
		},
		Desc:          field.Desc.GetLang(lang),
		Placeholder:   field.Placeholder.GetLang(lang),
		AdvancedField: field.internalAdvancedField.ToLang(lang),
	}
	return
}

type internalInputFormField struct {
	Key  string    `json:"key,omitempty"`
	Type FieldType `json:"type,omitempty"`

	Label       Lang `json:"label,omitempty"`
	Desc        Lang `json:"desc,omitempty"`
	Placeholder Lang `json:"placeholder,omitempty"`

	internalAdvancedField
}

type internalUIConfig struct {
	uiConfig

	// RawDisplay is the raw display configuration.
	RawDisplay Display `json:"$display"`
	// Display is a syntax sugar for RawDisplay.
	// internalDisplay implements UnmarshalJSON, support two kinds of syntax sugars.
	// {"key": "value"} will be converted to one displayConditionEquals,
	// {"key": ["v1", "v2"]} will be converted to one displayConditionIn.
	Display    *internalDisplay    `json:"display"`
	SelectFrom *internalSelectFrom `json:"selectFrom,omitempty"`
}

type uiConfig struct {
	Component          string      `json:"component,omitempty"`
	SelectFrom         *selectFrom `json:"selectFrom,omitempty"`
	Display            *Display    `json:"display,omitempty"`
	DateFormat         string      `json:"dateFormat,omitempty"`
	DisableCustomInput bool        `json:"disableCustomInput,omitempty"`
	Multiple           bool        `json:"multiple,omitempty"`
	Disabled           bool        `json:"disabled,omitempty"`
	Copyable           bool        `json:"copyable,omitempty"`
}

func (c uiConfig) deepCopy() *uiConfig {
	return &uiConfig{
		Component:          c.Component,
		DateFormat:         c.DateFormat,
		Multiple:           c.Multiple,
		Disabled:           c.Disabled,
		Copyable:           c.Copyable,
		DisableCustomInput: c.DisableCustomInput,
	}
}

type selectFromSource string

// keep these unused code.
const (
	selectFromStatic  selectFromSource = "static"  // nolint
	selectFromAdapter selectFromSource = "adapter" // nolint
)

type internalSelectFrom struct {
	selectFrom

	Options internalOptions `json:"options"`
}

func (f internalSelectFrom) ToLang(lang string) (result *selectFrom) {
	if f.Source == "" {
		return nil
	}
	result = &f.selectFrom
	if len(f.Options) > 0 {
		result.Options = f.Options.ToLangOptions(lang)
	}
	return
}

type selectFrom struct {
	Source selectFromSource `json:"source,omitempty"`
	// only valid for Source == selectFromStatic
	Options Options `json:"options,omitempty"`
	// the following fields are only valid for Source == selectFromAdapter
	Class        string `json:"class,omitempty"`
	EnableSearch bool   `json:"enableSearch,omitempty"`
	EnablePage   bool   `json:"enablePage,omitempty"`
}

type internalDisplay struct {
	value Display
}
type Display [][]DisplayCondition
type DisplayCondition struct {
	Key       string                    `json:"key,omitempty"`
	Operation displayConditionOperation `json:"operation,omitempty"`
	Value     any                       `json:"value,omitempty"`
}
type displayConditionOperation string

const (
	displayConditionEquals displayConditionOperation = "equals"
	displayConditionIn     displayConditionOperation = "in"
	displayConditionNotIn  displayConditionOperation = "not_in" // nolint: varcheck
)

func (src *internalDisplay) UnmarshalJSON(data []byte) (err error) {
	display := Display{}
	defer func() {
		src.value = display
	}()
	err = json.Unmarshal(data, &display)
	if err == nil {
		return
	}
	displayMap := map[string]any{}
	err = json.Unmarshal(data, &displayMap)
	if err != nil {
		return errors.New("value configuration invalid") // TODO: add documentation link for this.
	}

	// if value defined as a map like:
	// {"key1": "value1", "key2": "value2", "key3": ["value3", "value4"]}
	// then transform to map to standard value:
	// [[
	// 	{"key": "key1", "operation": "equals", "value": "value1"},
	// 	{"key": "key2", "operation": "equals", "value": "value2"},
	// 	{"key": "key3", "operation": "in", "value": ["value3", "value4"]}
	// ]]
	if len(displayMap) == 0 {
		return
	}
	var conditions []DisplayCondition
	for key, value := range displayMap {
		if key == "" {
			return errors.New("value map key can't be empty")
		}
		operation := displayConditionEquals
		if _, ok := value.([]any); ok {
			operation = displayConditionIn
		}
		conditions = append(conditions, DisplayCondition{
			Key:       key,
			Operation: operation,
			Value:     value,
		})
	}
	display = append(display, conditions)
	return nil
}

// BaseField defines the base fields of input form and output schema.
type BaseField struct {
	Key   string    `json:"key,omitempty"`
	Label string    `json:"label,omitempty"`
	Type  FieldType `json:"type,omitempty"`
}

type internalAdvancedField struct {
	AdvancedField

	UI     *internalUIConfig       `json:"ui"`
	Child  *internalInputFormField `json:"child"`
	Fields internalInputFormFields `json:"fields"`
}

func (f internalAdvancedField) ToLang(lang string) AdvancedField {
	result := f.AdvancedField
	if f.Child != nil {
		result.Child = f.Child.ToLang(lang)
	}
	if len(f.Fields) > 0 {
		result.Fields = f.Fields.ToLang(lang)
	}
	if f.UI != nil {
		result.UI = f.UI.uiConfig.deepCopy()
		if f.UI.SelectFrom != nil {
			result.UI.SelectFrom = f.UI.SelectFrom.ToLang(lang)
		}
		if len(f.UI.RawDisplay) > 0 {
			result.UI.Display = &f.UI.RawDisplay
		} else if f.UI.Display != nil {
			result.UI.Display = &f.UI.Display.value
		}
	}
	return result
}

type AdvancedField struct {
	Child     *InputFormField `json:"child,omitempty"`
	Fields    InputFormFields `json:"fields,omitempty"`
	UI        *uiConfig       `json:"ui,omitempty"`
	Default   any             `json:"default,omitempty"`
	Required  bool            `json:"required,omitempty"`
	Encrypted bool            `json:"encrypted,omitempty"`
}

type (
	// InputFormFields defines input form for building ui.
	InputFormFields []*InputFormField
	InputFormField  struct {
		BaseField

		Desc        string `json:"desc,omitempty"`
		Placeholder string `json:"placeholder,omitempty"`

		AdvancedField
	}

	Option struct {
		// ID is optional, if multiple values are sample, can specify a unique ID for each option.
		ID    string `json:"id,omitempty"`
		Label string `json:"label"`
		Value any    `json:"value"`
		Image string `json:"image,omitempty"`
	}

	Options []Option

	internalOption struct {
		Option

		Label Lang `json:"label"`
		Value any  `json:"value"`
	}
	internalOptions []internalOption
)

func (options internalOptions) ToDefault() (result Options) {
	return options.ToLangOptions(defaultLanguage)
}

func (options internalOptions) ToLangOptions(lang string) (result Options) {
	result = make(Options, len(options))
	for i, option := range options {
		result[i] = Option{
			ID:    option.ID,
			Image: option.Image,
			Label: option.Label.GetLang(lang),
			Value: option.Value,
		}
	}
	return result
}

var AnySchema = InputFormFields{
	&InputFormField{
		BaseField: BaseField{
			Key: "If use as input schema, should handle every field by yourself",
		},
	},
}

func IsAnySchema(fields InputFormFields) bool {
	if len(fields) != len(AnySchema) {
		return false
	}
	for i, field := range fields {
		if field != AnySchema[i] {
			return false
		}
	}
	return true
}

type FieldType string

const (
	StructFieldType FieldType = "struct"
	ListFieldType   FieldType = "list"
	IntFieldType    FieldType = "integer"
	FloatFieldType  FieldType = "float"
	BoolFieldType   FieldType = "bool"
	StringFieldType FieldType = "string"
)

func unmarshalKey(key string, data map[string]json.RawMessage, output interface{}) error {
	if _, found := data[key]; found {
		if err := json.Unmarshal(data[key], output); err != nil {
			return fmt.Errorf("failed to unmarshall key %q with data[%q]", key, data[key])
		}
	}
	return nil
}

func unmarshalString(data []byte) (string, error) {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	return value, nil
}

type (
	// DEPRECATED: will be removed
	OutputFields []*OutputField

	OutputField struct {
		BaseField

		Child  *OutputField `json:"child,omitempty"`
		Where  []Where      `json:"where"`
		Fields OutputFields `json:"fields,omitempty"`
	}

	Where string
)

type internalOutputFields []*internalOutputField

// ToDefault returns the default english language version for each value field.
func (fields internalOutputFields) ToDefault() (result OutputFields) {
	return fields.ToLang(defaultLanguage)
}

func (fields internalOutputFields) ToLang(lang string) (result OutputFields) {
	result = make(OutputFields, len(fields))
	for i, field := range fields {
		result[i] = field.ToLangOutputField(lang)
	}
	return result
}

type internalOutputField struct {
	Key    string               `json:"key,omitempty"`
	Type   FieldType            `json:"type,omitempty"`
	Label  Lang                 `json:"label,omitempty"`
	Child  *internalOutputField `json:"child,omitempty"`
	Where  []Where              `json:"where"`
	Fields internalOutputFields `json:"fields,omitempty"`
}

func (field internalOutputField) ToDefaultOutputField() *OutputField {
	var child *OutputField
	if field.Child != nil {
		child = field.Child.ToDefaultOutputField()
	}

	return &OutputField{
		BaseField: BaseField{
			Key:   field.Key,
			Label: field.Label.EN,
			Type:  field.Type,
		},
		Child:  child,
		Where:  field.Where,
		Fields: field.Fields.ToDefault(),
	}
}

func (field internalOutputField) ToLangOutputField(lang string) *OutputField {
	var child *OutputField
	if field.Child != nil {
		child = field.Child.ToLangOutputField(lang)
	}

	return &OutputField{
		BaseField: BaseField{
			Key:   field.Key,
			Label: field.Label.GetLang(lang),
			Type:  field.Type,
		},
		Child:  child,
		Where:  field.Where,
		Fields: field.Fields.ToLang(lang),
	}
}

const (
	WhereTemplate Where = "template"
	WhereForeach  Where = "foreach"
)
