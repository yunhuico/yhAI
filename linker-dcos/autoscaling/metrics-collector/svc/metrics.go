package svc

import (
	"fmt"
	"log"

	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/constant"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/daemon"
)

func ResultMetrics() (metricsText string, err error) {
	// log.Println("retrieving metrics and calculating ...")

	if daemon.D == nil || !daemon.D.IsRunning() {
		log.Println("daemon instance is not running")
		return
	}

	resultMetrics, err := daemon.D.ResultMetrics()
	if err != nil {
		log.Printf("get result metrics error: %v\n", err)
		return
	}

	for _, line := range resultMetrics.Lines {
		metricsText += line
	}

	return
}

func HostMetrics() (hostMetricsText string, err error) {
	// log.Println("retrieving host metrics ...")
	if daemon.D == nil || !daemon.D.IsRunning() {
		log.Println("daemon instance is not running")
		return
	}
	if !daemon.D.Config.HostMonitorEnabled {
		return
	}
	// enabled
	hostMetrics, err := daemon.D.HostMetrics()
	if err != nil {
		log.Printf("get host metrics error: %v\n", err)
		return
	}
	// append comments
	hostMetricsText += fmt.Sprintf("# TYPE %s gauge\n", constant.IndexHostCPUUsage)
	hostMetricsText += fmt.Sprintf("# TYPE %s gauge\n", constant.IndexHostMemUsage)
	// convert metrics structs to strings
	for _, m := range hostMetrics {
		for _, l := range m.Lines {
			hostMetricsText += l.String()
		}
	}
	return
}
