package adapter

import (
	"fmt"
)

var (
	adapterManager *AdapterManager
)

// init cost much time, so just init adapter when actual require.
func init() {
	adapterManager = NewAdapterManager()
}

func NewAdapterManager() *AdapterManager {
	return &AdapterManager{
		metaMap: map[string]*Meta{},
		specMap: map[string]*Spec{},
	}
}

// GetAdapterManager return the global adapter manager.
func GetAdapterManager() *AdapterManager {
	return adapterManager
}

func MustLookupSpec(class string) *Spec {
	spec := adapterManager.LookupSpec(class)
	if spec == nil {
		panic(fmt.Sprintf("class %q not found", class))
	}
	return spec
}
