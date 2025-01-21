package yaml

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// UnmarshalWithBytes unmarshal dist with data
func UnmarshalWithBytes(data []byte, dist interface{}) error {
	err := yaml.Unmarshal(data, dist)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return nil
}

// UnmarshalWithFile unmarshal yaml file to a dist object
func UnmarshalWithFile(file string, dist interface{}) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read yaml %s: %w", file, err)
	}
	err = UnmarshalWithBytes(data, dist)
	if err != nil {
		return fmt.Errorf("unmarshal yaml %s: %w", file, err)
	}

	return nil
}
