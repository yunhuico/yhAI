package daemon

import (
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/constant"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

type Avg struct {
	avgCPUHigh float64
	avgCPULow  float64
	avgMemHigh float64
	avgMemLow  float64
}

func avg(metrics types.Metrics) (a *Avg) {
	a = &Avg{}

	var sumCPUHigh, sumCPULow, sumMemHigh, sumMemLow float64
	var numCPUHigh, numCPULow, numMemHigh, numMemLow int
	for _, line := range metrics.Lines {
		index := line.Index
		switch {
		case isCPUHigh(index):
			sumCPUHigh += line.FloatVar
			numCPUHigh++
		case isCPULow(index):
			sumCPULow += line.FloatVar
			numCPULow++
		case isMemHigh(index):
			sumMemHigh += line.FloatVar
			numMemHigh++
		case isMemLow(index):
			sumMemLow += line.FloatVar
			numMemLow++
		default:
		}

		// 1.0 is the ignored float value in metrics
		ignoreVar := 1.0

		a.avgCPUHigh = ignoreVar
		if numCPUHigh != 0 && sumCPUHigh != 0 {
			a.avgCPUHigh = sumCPUHigh / float64(numCPUHigh)
		}
		a.avgCPULow = ignoreVar
		if numCPULow != 0 && sumCPULow != 0 {
			a.avgCPULow = sumCPULow / float64(numCPULow)
		}
		a.avgMemHigh = ignoreVar
		if numMemHigh != 0 && sumMemHigh != 0 {
			a.avgMemHigh = sumMemHigh / float64(numMemHigh)
		}
		a.avgMemLow = ignoreVar
		if numMemLow != 0 && sumMemLow != 0 {
			a.avgMemLow = sumMemLow / float64(numMemLow)
		}
	}
	return
}

func isCPUHigh(index string) bool {
	return index == constant.IndexCPUHigh
}

func isCPULow(index string) bool {
	return index == constant.IndexCPULow
}

func isMemHigh(index string) bool {
	return index == constant.IndexMemHigh
}

func isMemLow(index string) bool {
	return index == constant.IndexMemLow
}
