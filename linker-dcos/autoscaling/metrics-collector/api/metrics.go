package api

import (
	"log"

	"gopkg.in/kataras/iris.v6"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/svc"
)

// GetRoot handles GET /metrics
func GetMetrics(ctx *iris.Context) {
	metricsText, err := svc.ResultMetrics()
	if err != nil {
		log.Printf("get NFV metrics error: %v", err)
		ctx.Text(iris.StatusInternalServerError, err.Error())
		return
	}

	hostMetricsText, err := svc.HostMetrics()
	if err != nil {
		log.Printf("get host metrics error: %v\n", err)
		ctx.Text(iris.StatusInternalServerError, err.Error())
		return
	}
	ctx.Text(iris.StatusOK, metricsText+hostMetricsText)
	return
}
