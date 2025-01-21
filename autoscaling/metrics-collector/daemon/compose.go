package daemon

import (
	"fmt"

	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/constant"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

// compose() combines Map into types.Line structures
// and then 'marshal' the structures to metrics lines
func compose(avgMap map[string]Avg, metricsMap map[string]types.Metrics) (composedMetrics *types.RawMetrics) {
	composedMetrics = &types.RawMetrics{}

	composedMetrics.Lines = append(composedMetrics.Lines, fmt.Sprintf("# TYPE %s gauge\n", constant.IndexCPUAvgHigh))
	composedMetrics.Lines = append(composedMetrics.Lines, fmt.Sprintf("# TYPE %s gauge\n", constant.IndexCPUAvgLow))
	composedMetrics.Lines = append(composedMetrics.Lines, fmt.Sprintf("# TYPE %s gauge\n", constant.IndexMemAvgHigh))
	composedMetrics.Lines = append(composedMetrics.Lines, fmt.Sprintf("# TYPE %s gauge\n", constant.IndexMemAvgLow))
	for appID, avg := range avgMap {
		m := make(map[string]string)
		// copy fields from then first metrics line of an app
		originMap := metricsMap[appID].Lines[0].Map
		m[constant.KeyAppID] = appID
		m[constant.KeyAlert] = originMap[constant.KeyAlert]
		m[constant.KeyAppContainerID] = originMap[constant.KeyAppContainerID]
		m[constant.KeyGroupID] = originMap[constant.KeyGroupID]
		m[constant.KeyImage] = originMap[constant.KeyImage]
		m[constant.KeyRepairTemplateID] = originMap[constant.KeyRepairTemplateID]
		m[constant.KeyServiceGroupID] = originMap[constant.KeyServiceGroupID]
		m[constant.KeyServiceGroupInstanceID] = originMap[constant.KeyServiceGroupInstanceID]
		m[constant.KeyServiceOrderID] = originMap[constant.KeyServiceOrderID]

		m[constant.KeyAlertName] = constant.HighCpuAlert
		composedMetrics.Lines = append(composedMetrics.Lines, join(constant.IndexCPUAvgHigh, m, avg.avgCPUHigh))

		m[constant.KeyAlertName] = constant.LowCpuAlert
		composedMetrics.Lines = append(composedMetrics.Lines, join(constant.IndexCPUAvgLow, m, avg.avgCPULow))

		m[constant.KeyAlertName] = constant.HighMemoryAlert
		composedMetrics.Lines = append(composedMetrics.Lines, join(constant.IndexMemAvgHigh, m, avg.avgMemHigh))

		m[constant.KeyAlertName] = constant.LowMemoryAlert
		composedMetrics.Lines = append(composedMetrics.Lines, join(constant.IndexMemAvgLow, m, avg.avgMemLow))
	}
	return
}

func join(index string, m map[string]string, f float64) string {
	line := &types.Line{
		Index:    index,
		Map:      m,
		FloatVar: f,
	}
	return line.String()
}
