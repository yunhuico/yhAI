package daemon

import (
	"fmt"
	"log"

	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/constant"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/env"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/util"
)

var (
	// D is the running instance of daemon
	D *MetricsDaemon
)

// start shared instance of metrics daemon
func StartDaemonInstance() (err error) {
	cadvisors, _ := env.Get(constant.EnvCADVISORS).ToStringArr(constant.AddrSeparator)
	daemonMode, _ := env.Get(constant.EnvDaemonMode).ToString()
	pollingSec, _ := env.Get(constant.EnvPollingSec).ToInt()
	cadvisorTimeout, _ := env.Get(constant.EnvCadvisorTimeout).ToInt()

	log.Printf("env %s: %v\n", constant.EnvCADVISORS, cadvisors)
	log.Printf("env %s: %v\n", constant.EnvDaemonMode, daemonMode)
	log.Printf("env %s: %v\n", constant.EnvPollingSec, pollingSec)
	log.Printf("env %s: %v\n", constant.EnvCadvisorTimeout, cadvisorTimeout)

	updaterEnabled, _ := env.Get(constant.EnvEnableUpdater).ToBool()
	addrUpdateSec, _ := env.Get(constant.EnvAddrUpdateSec).ToInt()
	mesosEndpoint, _ := env.Get(constant.EnvMesosEndpoint).ToString()
	cadvisorPort, _ := env.Get(constant.EnvCadvisorPort).ToInt()
	hostMonitorEnabled, _ := env.Get(constant.EnvEnableHostMonitor).ToBool()

	log.Printf("env %s: %v\n", constant.EnvEnableUpdater, updaterEnabled)
	log.Printf("env %s: %v\n", constant.EnvAddrUpdateSec, addrUpdateSec)
	log.Printf("env %s: %v\n", constant.EnvMesosEndpoint, mesosEndpoint)
	log.Printf("env %s: %v\n", constant.EnvCadvisorPort, cadvisorPort)

	updaterConfig := AddrUpdaterConfig{
		IntervalSec:   addrUpdateSec,
		MesosEndpoint: mesosEndpoint,
		UpdateOnStart: true,
		TimeoutMs:     2000,
		CadvisorPort:  cadvisorPort,
	}
	addrUpdater, err := NewAddrUpdater(updaterConfig)
	if err != nil {
		log.Printf("new AddrUpdater error: %v\n", err)
		return
	}

	config := MetricsDaemonConfig{
		Mode:               daemonMode,
		Cadvisors:          cadvisors,
		PollingSec:         pollingSec,
		CadvisorTimeout:    cadvisorTimeout,
		UpdaterEnabled:     updaterEnabled,
		Updater:            addrUpdater,
		HostMonitorEnabled: hostMonitorEnabled,
	}

	daemon, err := NewMetricsDaemon(config)
	if err != nil {
		log.Printf("new metrics daemon error: %v\n", err)
		return
	}
	fmt.Println("Metrics daemon will start with config: ")
	util.PrintPretty(daemon.Config)
	err = daemon.Start()
	if err != nil {
		log.Printf("start metrics daemon error: %v\n", err)
		return
	}
	D = daemon
	return
}
