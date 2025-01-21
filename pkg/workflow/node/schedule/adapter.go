package schedule

import (
	"embed"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/common"
)

const className = "ultrafox/schedule"

var adapterClass = common.AdapterClass(className)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapterMeta := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapterMeta.RegisterSpecsByDir(adapterDir)
}
