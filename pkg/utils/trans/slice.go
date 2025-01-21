package trans

import (
	"fmt"
	"reflect"

	"github.com/spf13/cast"
)

// ToAnySlice trans to any slice.
func ToAnySlice(input any) (result []any, err error) {
	if input == nil {
		return []any{}, nil
	}

	var ok bool
	result, ok = input.([]any)
	if ok {
		return result, nil
	}

	refValue := reflect.ValueOf(input)
	if refValue.Kind() == reflect.Ptr {
		refValue = refValue.Elem()
	}

	if kind := refValue.Kind(); kind != reflect.Slice && kind != reflect.Array {
		err = fmt.Errorf("cannot convert input to slice or array")
		return
	}

	result = make([]any, refValue.Len())
	for i := 0; i < refValue.Len(); i++ {
		result[i] = refValue.Index(i).Interface()
	}
	return result, nil
}

// ToStringSlice try to transform []any.
func ToStringSlice(input any) (result []string, err error) {
	if input == nil {
		return
	}

	refValue := reflect.ValueOf(input)
	if refValue.Kind() == reflect.Ptr {
		refValue = refValue.Elem()
	}
	if refValue.Kind() != reflect.Slice {
		err = fmt.Errorf("input must be a slice")
		return
	}

	result = make([]string, refValue.Len())
	for i := 0; i < refValue.Len(); i++ {
		result[i], err = cast.ToStringE(refValue.Index(i).Interface())
		if err != nil {
			err = fmt.Errorf("input[%d] cannot transform to string: %w", i, err)
			return
		}
	}

	return
}
