package daemon

import (
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/constant"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

// group() classify metrics by App ID (component ID)
func group(metricsArr []types.Metrics) (m map[string]types.Metrics) {
	m = make(map[string]types.Metrics)
	for _, metrics := range metricsArr {
		for _, line := range metrics.Lines {
			k := line.Map[constant.KeyAppID]
			v := m[k]
			v.Lines = append(v.Lines, line)
			m[k] = v
		}
	}
	return
}
