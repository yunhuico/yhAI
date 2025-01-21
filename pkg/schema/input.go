package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/spf13/cast"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/template"
)

type FieldValueReader interface {
	Read(value string) (string, error)
}

type OriginValueRender struct{}

func (o OriginValueRender) Read(value string) (string, error) {
	return value, nil
}

func NewOriginValueRender() FieldValueReader {
	return &OriginValueRender{}
}

type mapScope map[string]any

func (m mapScope) GetScopeData() map[string]any {
	return m
}

type MapReader struct {
	template *template.Engine
}

func (m MapReader) Read(origin string) (actual string, err error) {
	b, err := m.template.RenderTemplate(origin)
	if err != nil {
		err = fmt.Errorf("render template: %w", err)
		return
	}
	actual = string(b)
	return
}

func NewMapReader(data map[string]any) FieldValueReader {
	return &MapReader{
		template: template.NewTemplateEngine(mapScope(data)),
	}
}

// RenderJSON renders the schema to json, is a field render error but it's required, ignore error.
// the result will discard the error field
func RenderJSON(fields adapter.InputFormFields, inputFields map[string]any, render FieldValueReader) ([]byte, error) {
	result, err := RenderInputFieldsBySchema(fields, inputFields, render)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

// RenderInputFieldsBySchema renders the schema to map.
//
// if a field render error throws the error only the field is required.
func RenderInputFieldsBySchema(fields adapter.InputFormFields, inputFields map[string]any, reader FieldValueReader) (dest map[string]any, err error) {
	dest = map[string]any{}
	for _, field := range fields {
		originValue, ok := inputFields[field.Key]
		if !ok {
			continue
		}

		if field.Type == adapter.ListFieldType {
			var actualValueList []any
			// TODO(sword): check field Child should define when checking the adapter definition.
			if field.Child == nil {
				err = fmt.Errorf("field %s's child is nil", field.Key)
				return
			}

			if field.Child.Type == adapter.StructFieldType {
				var structCollection []map[string]any
				structCollection, err = transformToMapArray(originValue)
				if err != nil {
					err = fmt.Errorf("transforming value mapArray: %w", err)
					return
				}

				for _, structItem := range structCollection {
					var actualValueItem map[string]any
					actualValueItem, err = RenderInputFieldsBySchema(field.Child.Fields, structItem, reader)
					if err != nil {
						break
					}
					actualValueList = append(actualValueList, actualValueItem)
				}
			} else {
				originValueList, ok := originValue.([]any)
				if !ok {
					err = fmt.Errorf("expect field %s is [string], but actual is not", field.Key)
					return
				}
				for i, originValue := range originValueList {
					var actualValueItem any
					actualValueItem, err = calcField(fmt.Sprintf("%s[%d]", field.Key, i), originValue, field.Child.Type, reader)
					if err != nil {
						// skip current one
						break
					}
					actualValueList = append(actualValueList, actualValueItem)
				}
			}
			if err != nil {
				err = nil // ignore this error, skip this field.
				continue
			}
			dest[field.Key] = actualValueList
			continue
		}

		var actualValue any
		// ignore this error, if calculated failed, use default value.
		actualValue, err = calcField(field.Key, originValue, field.Type, reader)
		if err != nil {
			err = nil // ignore this error, skip this field.
			continue
		}
		dest[field.Key] = actualValue
	}
	return
}

func transformToMapArray(originValue any) ([]map[string]any, error) {
	var result []map[string]any
	switch val := originValue.(type) {
	case []any:
		for _, rv := range val {
			switch innerVal := rv.(type) {
			case map[string]any:
				result = append(result, innerVal)
			default:
				return nil, errors.New("unsupported type")
			}
		}
	case []map[string]any:
		return val, nil
	default:
		return nil, errors.New("unsupported type")
	}
	return result, nil
}

func parseValue(str string, kind adapter.FieldType) (v interface{}, err error) {
	switch kind {
	case adapter.BoolFieldType:
		return cast.ToBoolE(str)
	case adapter.IntFieldType:
		v, err = cast.ToIntE(str)
		if err != nil {
			f, err1 := strconv.ParseFloat(str, 64)
			// example: str is "40.00", can't trans to int directly.
			// try to convert to float64 first.
			if err1 != nil {
				return
			}

			// check whether accuracy is lost
			if math.Trunc(f) != f {
				// if exists loss of precision, failed to trans.
				return
			}

			v = int(f)
			err = nil
			return
		}
		return
	case adapter.FloatFieldType:
		return cast.ToFloat64E(str)
	case adapter.StringFieldType:
		return cast.ToStringE(str)
	}
	return nil, fmt.Errorf("unsupported kind %q", kind)
}

func getReflectType(value any) reflect.Kind {
	return reflect.ValueOf(value).Kind()
}

func calcField(field string, value any, expectedKind adapter.FieldType, render FieldValueReader) (any, error) {
	valueType := getReflectType(value)
	if valueType != reflect.String {
		switch valueType {
		case reflect.Int:
			fallthrough
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			fallthrough
		case reflect.Uint:
			fallthrough
		case reflect.Uint8:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			fallthrough
		case reflect.Uint64:
			if expectedKind == adapter.StringFieldType {
				return cast.ToStringE(value)
			}
			if expectedKind == adapter.FloatFieldType {
				return cast.ToFloat64E(value)
			}
			if expectedKind != adapter.IntFieldType {
				return nil, fmt.Errorf("field %q is not int type", field)
			}
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			if expectedKind == adapter.StringFieldType {
				return cast.ToStringE(value)
			}
			if expectedKind == adapter.IntFieldType {
				return cast.ToIntE(value)
			}
			if expectedKind != adapter.FloatFieldType {
				return nil, fmt.Errorf("field %q is not float type", field)
			}
		case reflect.Bool:
			if expectedKind == adapter.StringFieldType {
				return cast.ToStringE(value)
			}
			if expectedKind != adapter.BoolFieldType {
				return nil, fmt.Errorf("field %q is not bool type", field)
			}
		default:
			return nil, fmt.Errorf("field %q is not supported type <%q>", field, valueType.String())
		}
		return value, nil
	}

	valueStr, err := render.Read(value.(string))
	if err != nil {
		return nil, fmt.Errorf("get field %q value error: %w", field, err)
	}
	value, err = parseValue(valueStr, expectedKind)
	if err != nil {
		return nil, fmt.Errorf("parse field %q value <%q> error: %w", field, value, err)
	}
	return value, nil
}
