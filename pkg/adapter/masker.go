package adapter

import (
	"strings"
)

type Masker struct {
	// fieldsMaskFunc which func to use depends on the field style.
	fieldsMaskFunc map[string]maskStringFunc
}

func NewMasker(fields InputFormFields) Masker {
	fns := map[string]maskStringFunc{}
	for _, field := range fields {
		if field.Encrypted {
			fns[field.Key] = encryptField
		}
	}
	return Masker{
		fieldsMaskFunc: fns,
	}
}

// Mask no side effect on input.
func (m Masker) Mask(inputFields map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range inputFields {
		if f, ok := m.fieldsMaskFunc[k]; ok {
			result[k] = f(v)
		} else {
			result[k] = v
		}
	}
	return result
}

// MergeChangedValue to currentFields, requestFields's value is masked if it's not change.
func (m Masker) MergeChangedValue(currentFields map[string]string, requestFields map[string]string) (result map[string]string, changed bool) {
	result = map[string]string{}
	currentMask := m.Mask(currentFields)

	for k, v := range requestFields {
		result[k] = v
	}
	for k, encryptedValue := range currentMask {
		if encryptedValue != result[k] { // user changed this field's value
			changed = true
		} else {
			result[k] = currentFields[k]
		}
	}
	return
}

type maskStringFunc func(string) string

// encryptField
//   - if input length <= 9, every character replace by '*'
//   - else the 3 front and tail characters keep original, otherwise replace by '*'
func encryptField(input string) string {
	if input == "" {
		return ""
	}
	l := len(input)
	if l <= 9 {
		return strings.Repeat("*", l)
	}

	return input[:3] + strings.Repeat("*", l-6) + input[l-3:]
}
