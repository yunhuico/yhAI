package validator

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Validate validates a struct or a slice of struct in go-playground/validator's way.
// Ref: https://pkg.go.dev/github.com/go-playground/validator/v10
func Validate(value any) (err error) {
	valid := validator.New()
	// go-playground/validator does not automatically distinguish between
	// struct and slice of struct.
	// https://github.com/go-playground/validator/issues/595
	switch k := solidValue(reflect.ValueOf(value)).Kind(); k {
	case reflect.Slice:
		err = valid.Var(value, "dive")
	case reflect.Struct:
		err = valid.Struct(value)
	default:
		err = fmt.Errorf("unexpected reflection kind %s of value", k.String())
		return
	}
	if err == nil {
		// everything is fine
		return
	}

	var ok bool
	// validator has a special way of error definition
	// https://github.com/go-playground/validator#error-return-value
	err, ok = err.(validator.ValidationErrors)
	if !ok {
		err = fmt.Errorf("validation error: %w", err)
		return
	}

	err = fmt.Errorf("validating value: %w", err)
	return
}

// solidValue traverses the pointer chain to fetch
// the solid value
func solidValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}
