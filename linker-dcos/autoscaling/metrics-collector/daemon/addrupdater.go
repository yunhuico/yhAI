package daemon

import (
	"errors"
	"fmt"
	"log"
)

const (
	// update cadvisor addresses every 5min by default
	defaultIntervalSec   = 5 * 60
	defaultMesosEndpoint = "master.mesos/mesos"
	defaultTimeoutMs     = 2000
	defaultCadvisorPort  = 10000
)

var (
	// ErrEmptyHostname returned when mesos API return an empty response
	// or it is not parsed correctlly
	ErrEmptyHostname = errors.New("got empty hostnames")
)

// AddrUpdater is an Updater which will refresh cadvisors addresses
type AddrUpdater struct {
	Config AddrUpdaterConfig
}

type AddrUpdaterConfig struct {
	IntervalSec   int
	MesosEndpoint string
	UpdateOnStart bool
	TimeoutMs     int
	CadvisorPort  int
}

func NewAddrUpdater(config AddrUpdaterConfig) (addrUpdater AddrUpdater, err error) {
	if config.IntervalSec <= 0 {
		config.IntervalSec = defaultIntervalSec
	}
	if config.MesosEndpoint == "" {
		config.MesosEndpoint = defaultMesosEndpoint
	}
	if config.TimeoutMs <= 0 {
		config.TimeoutMs = defaultTimeoutMs
	}
	if config.CadvisorPort <= 0 {
		config.CadvisorPort = defaultCadvisorPort
	}

	addrUpdater.Config = config
	return
}

func (u AddrUpdater) Update(config MetricsDaemonConfig) (MetricsDaemonConfig, error) {
	hostnames, err := MesosHosts(u.Config.MesosEndpoint, u.Config.TimeoutMs)
	if err != nil {
		return config, err
	}
	if len(hostnames) == 0 {
		return config, ErrEmptyHostname
	}

	var cadvisors []string
	for _, h := range hostnames {
		c := fmt.Sprintf("%s:%d", h, u.Config.CadvisorPort)
		cadvisors = append(cadvisors, c)
	}
	log.Printf("got cadvisors: %v\n", cadvisors)
	config.Cadvisors = cadvisors
	return config, nil
}

func (u AddrUpdater) IntervalSec() int {
	return u.Config.IntervalSec
}

func (u AddrUpdater) UpdateOnStart() bool {
	return u.Config.UpdateOnStart
}
