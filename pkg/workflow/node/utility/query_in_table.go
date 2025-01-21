package utility

import (
	"embed"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type QueryInTable struct {
	FieldKey     string      `json:"fieldKey"`
	QueryTable   []TableItem `json:"queryTable"`
	DefaultValue string      `json:"defaultValue"`
}

type TableItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (q QueryInTable) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/utility#queryInTable")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(QueryInTable)
		},
		InputForm: spec.InputSchema,
	}
}

func (q QueryInTable) Run(_ *workflow.NodeContext) (result any, err error) {
	var matchValue string
	defer func() {
		if err != nil {
			return
		}
		result = map[string]any{
			"data": matchValue,
		}
	}()

	if len(q.QueryTable) == 0 {
		matchValue = q.DefaultValue
		return
	}

	for _, item := range q.QueryTable {
		if item.Key == q.FieldKey {
			matchValue = item.Value
			return
		}
	}

	matchValue = q.DefaultValue
	return
}

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapter := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapter.RegisterSpecsByDir(adapterDir)

	workflow.RegistryNodeMeta(&QueryInTable{})
}
