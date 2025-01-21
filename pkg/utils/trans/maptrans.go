package trans

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// StructToMap trans struct to map.
func StructToMap(input any) (map[string]any, error) {
	result := map[string]any{}
	config := &mapstructure.DecoderConfig{
		Result:  &result,
		TagName: "json",
		Squash:  true,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil { // this error should never happen, but we should keep it here for safety.
		return nil, fmt.Errorf("failed to create mapstructure decoder: %w", err)
	}
	err = decoder.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode struct to map: %w", err)
	}
	return result, nil
}

func MapToStruct(input map[string]any, obj any) error {
	config := &mapstructure.DecoderConfig{
		Result:  obj,
		TagName: "json",
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return fmt.Errorf("failed to create mapstructure decoder: %w", err)
	}
	err = decoder.Decode(input)
	if err != nil {
		return fmt.Errorf("failed to decode map to struct: %w", err)
	}
	return nil
}

// TransformToNodeData to map[string]any if data is not basic type.
// TODO(sword): Precision loss Probably, make this function better.
func TransformToNodeData(input any) (output any, err error) {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	k := v.Kind()
	if (reflect.Bool <= k && k <= reflect.Complex128) || k == reflect.String {
		return input, nil
	}

	var b []byte
	if bytes, ok := input.([]byte); ok {
		if len(bytes) == 0 {
			return nil, nil
		}

		b = bytes
	} else {
		b, err = json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("marshal json when transform data: %v", err)
		}
	}
	var m any
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json when transform data: %v", err)
	}
	return m, nil
}
