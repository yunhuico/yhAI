package utils

import (
	"encoding/json"
	"fmt"
)

// FormatJSONIndent format json string to intend json
func FormatJSONIndent(msg json.RawMessage) (indentJSON string, err error) {
	var v interface{}
	err = json.Unmarshal(msg, &v)
	if err != nil {
		err = fmt.Errorf("format json indent: %w", err)
		return
	}
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		err = fmt.Errorf("format json indent: %w", err)
		return
	}
	return string(b), nil
}
