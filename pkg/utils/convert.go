package utils

import (
	"encoding/json"
	"fmt"
)

// ConvertStructToMap converts a struct into map.
func ConvertStructToMap(v any) (m map[string]any, err error) {
	marshaled, err := json.Marshal(v)
	if err != nil {
		err = fmt.Errorf("marshaling struct into JSON: %w", err)
		return
	}

	m = make(map[string]any)
	err = json.Unmarshal(marshaled, &m)
	if err != nil {
		err = fmt.Errorf("unmarhsling JSON into map: %w", err)
		return
	}

	return
}

// ConvertMapToStruct converts a map into struct.
func ConvertMapToStruct(m map[string]any, v any) (err error) {
	marshaled, err := json.Marshal(m)
	if err != nil {
		err = fmt.Errorf("marshaling map into JSON: %w", err)
		return
	}

	err = json.Unmarshal(marshaled, v)
	if err != nil {
		err = fmt.Errorf("unmarhsling JSON into %T: %w", v, err)
		return
	}

	return
}
