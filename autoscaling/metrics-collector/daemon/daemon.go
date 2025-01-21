package daemon

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/constant"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

const (
	// refresh NFV metrics every 5s by default
	defaultPullingSec = 5
	// length of metrics array cached in memory
	defaultCacheCount = 1
	// max length of cached metrics array
	maxCacheCount = 10
	// default daemon mode
	defaultDaemonMode = ModeOnRequest

	// ModePolling retrieves metrics every N seconds automaticlly
	ModePolling = "polling"
	// ModeOnRequest retrieves metrics when API /metrics is called
	ModeOnRequest = "onrequest"
)

var (
	// ErrInvalidConfig returned when 'Cadvisors' is not set correctly
	ErrInvalidCadvisors = errors.New("cadvisors must be set if updater not enabled")
	// ErrDaemonAlreadyRunning returned when daemon is already running
	ErrDaemonAlreadyRunning = errors.New("daemon is already running")
	// ErrFetchCadvisorMetrics returned when retrieve cAdvisor metrics error
	ErrFetchCadvisorMetrics = errors.New("fetch cadvisor metrics error")
	// ErrPollSecRequestMode returned when 'PollingSec' is set on mode 'OnRequest'
	ErrPollSecRequestMode = errors.New("polling second set on mode onrequest")
	// ErrUnsupport returned when 'Mode' is neither "", "polling" nor "onrequest"
	ErrUnsupportMode = errors.New("unsupported mode")
)

type Updater interface {
	Update(MetricsDaemonConfig) (MetricsDaemonConfig, error)
	UpdateOnStart() bool
	IntervalSec() int
}

type MetricsDaemon struct {
	isRunning       bool
	ticker          *time.Ticker
	Config          MetricsDaemonConfig
	resultMetrics   types.RawMetrics
	nfvMetrics      []types.Metrics
	nfvMetricsCache []types.Metrics
	hostMetrics     []types.Metrics
	mtx             *sync.Mutex
}

type MetricsDaemonConfig struct {
	// Mode is the policy how collector retrieves metrics
	// optional, using default (modePolling) if not set
	Mode string
	// Cadvisors are comma separated endpoints of cAdvisors
	// <IP>:<Port>;<IP>:<Port>
	Cadvisors []string
	// PollingSec is the interval that MtricsDaemon call cAdvisors every time
	// optional, using default if not set
	PollingSec int
	// CacheCount
	// optional, using default if not set
	CacheCount int
	// CadvisorTimeout is the response timeout of cAdvisor
	// optional, using default(5000) if not set
	// CadvisorTimeout/1000 must less than PollingSec
	CadvisorTimeout int
	// Updater is a tool to update daemon configurations while metrics daemon is running
	Updater Updater
	// UpdaterEnabled indicates whether updater is enabled
	UpdaterEnabled bool
	// HostMonitorEnabled indicates whether host machine CPU/Memory is monitored
	HostMonitorEnabled bool
}

func NewMetricsDaemon(config MetricsDaemonConfig) (daemon *MetricsDaemon, err error) {
	daemon = &MetricsDaemon{}

	if config.Mode == "" {
		config.Mode = defaultDaemonMode
	}
	switch config.Mode {
	case ModePolling:
		if config.PollingSec <= 0 {
			config.PollingSec = defaultPullingSec
		}
	case ModeOnRequest:
		// PollingSec is unnecessary on this mode
		if config.PollingSec > 0 {
			err = ErrPollSecRequestMode
			return
		}
	default:
		err = ErrUnsupportMode
		return
	}
	if !config.UpdaterEnabled {
		if len(config.Cadvisors) == 0 {
			err = ErrInvalidCadvisors
			return
		}
	}
	if config.CacheCount <= 0 {
		config.CacheCount = defaultCacheCount
	}
	daemon.Config = config
	daemon.mtx = &sync.Mutex{}
	return
}

func (d *MetricsDaemon) Start() error {
	if d.Config.UpdaterEnabled && d.Config.Updater != nil {
		d.startUpdater()
	}

	log.Printf("metrics daemon will run on mode: %s\n", d.Config.Mode)
	switch d.Config.Mode {
	case ModePolling:
		if d.isRunning {
			return ErrDaemonAlreadyRunning
		}
		return d.startCronJob()
	case ModeOnRequest:
		// On this mode, metrics daemon do not retrieve metrics automaticlly.
		// The daemon will passively refresh metrics when function
		// NFVMetrics() or ResultMetrics() is called.

		d.isRunning = true
		return nil
	default:
		// do nothing
	}
	return nil
}

func (d *MetricsDaemon) startUpdater() {
	updateFunc := func() {
		log.Println("updating daemon config ...")
		newConfig, err := d.Config.Updater.Update(d.Config)
		if err != nil {
			log.Printf("update config error: %v\n", err)
			// continue
		} else {
			d.Config = newConfig
		}
	}

	if d.Config.Updater.UpdateOnStart() {
		updateFunc()
	}

	updateTicker := time.NewTicker(time.Duration(d.Config.Updater.IntervalSec()) * time.Second)
	go func() {
		for range updateTicker.C {
			updateFunc()
		}
	}()
}

// start timer and continiously call cAdvisors' API to update metrics.
func (d *MetricsDaemon) startCronJob() error {
	pollingSec := d.Config.PollingSec
	log.Printf("daemon will fetch metrics every %ds\n", pollingSec)

	// get metrics immediately at the beginning
	fmt.Println("\n")
	d.refreshNFVMetrics()
	d.calcResultMetrics()
	if d.Config.HostMonitorEnabled {
		d.refreshHostMetrics()
	}

	d.ticker = time.NewTicker(time.Duration(pollingSec) * time.Second)
	go func() {
		for range d.ticker.C {
			// fmt.Println("\n")
			d.refreshNFVMetrics()
			d.calcResultMetrics()
			if d.Config.HostMonitorEnabled {
				d.refreshHostMetrics()
			}
		}
	}()

	d.isRunning = true
	return nil
}

func (d *MetricsDaemon) refreshNFVMetrics() (err error) {
	// 1. Retrieve filtered metrics from all cAdvisors in multiple goroutines
	// 2. Wait for all goroutines to finish, gather metrics to array
	indexs := []string{
		constant.IndexCPUHigh,
		constant.IndexCPULow,
		constant.IndexMemHigh,
		constant.IndexMemLow,
	}
	filter, err := NewNFVFilter(indexs)
	if err != nil {
		log.Printf("new NFV filter error: %v\n", err)
		return
	}
	rawMetricsArr, err := retrieveMultipleMetrics(d.Config.Cadvisors, d.Config.CadvisorTimeout, filter)
	if err != nil {
		log.Printf("retrieve multiple metrics error: %v\n", err)
		return
	}

	// 3. parse to struct
	var nfvMetrics []types.Metrics
	for _, rawMetrics := range rawMetricsArr {
		m, err := parseMetrics(rawMetrics)
		if err != nil {
			log.Printf("parse metrics error: %v\n", err)
			continue
		}
		if m != nil {
			nfvMetrics = append(nfvMetrics, *m)
		}
	}

	// 4. set daemon variables
	// d.mtx.Lock()
	// defer d.mtx.Unlock()
	d.nfvMetrics = nfvMetrics

	var lines int
	for _, r := range d.nfvMetrics {
		lines += len(r.Lines)
	}
	// log.Printf("len of filtered metrics lines: %d\n", lines)
	return nil
}

func (d *MetricsDaemon) refreshHostMetrics() (err error) {
	// 1. Retrieve filtered metrics from all cAdvisors in multiple goroutines
	// 2. Wait for all goroutines to finish, gather metrics to array
	metricsArr, err := retrieveMultipleHostMetrics(d.Config.Cadvisors, d.Config.CadvisorTimeout)
	if err != nil {
		log.Printf("retrieve multiple metrics error: %v\n", err)
		return
	}
	// fmt.Printf("host metrics: %+v\n", metricsArr)
	d.hostMetrics = metricsArr
	return nil
}

func (d *MetricsDaemon) calcResultMetrics() {
	// 5. Calculate result of average CPU and memory usage.
	// 6. Compose the result usage into new metrics for Prometheus.
	metricsMap := group(d.nfvMetrics)
	// log.Printf("metricsMap: %#v\n", metricsMap)

	var avgMap = make(map[string]Avg)
	for appID, v := range metricsMap {
		avgMap[appID] = *avg(v)
	}
	// log.Printf("avgMap: %+v\n", avgMap)
	for appID, v := range avgMap {
		log.Printf("AppID: %s, CPU High: %0.3f, CPU Low: %0.3f, Mem High: %0.3f, Mem Low: %0.3f",
			appID, v.avgCPUHigh, v.avgCPULow, v.avgMemHigh, v.avgMemLow)
	}

	var resultMetrics types.RawMetrics
	// add one line comment on top
	resultMetrics.Lines = append(resultMetrics.Lines, "# Generated by metrics-collector\n")

	composedMetrics := compose(avgMap, metricsMap)
	resultMetrics.Lines = append(resultMetrics.Lines, composedMetrics.Lines...)

	// log.Printf("resultMetrics: %+v\n", resultMetrics)

	d.resultMetrics = resultMetrics

	return
}

func (d *MetricsDaemon) NFVMetrics() (nfvMetrics []types.Metrics, err error) {
	if d.Config.Mode == ModeOnRequest {
		err = d.refreshNFVMetrics()
		if err != nil {
			return
		}
	}
	return d.nfvMetrics, nil
}

func (d *MetricsDaemon) HostMetrics() (metrics []types.Metrics, err error) {
	if !d.Config.HostMonitorEnabled {
		return
	}
	if d.Config.Mode == ModeOnRequest {
		err = d.refreshHostMetrics()
		if err != nil {
			return
		}
	}
	return d.hostMetrics, nil
}

func (d *MetricsDaemon) ResultMetrics() (resultMetrics types.RawMetrics, err error) {
	if d.Config.Mode == ModeOnRequest {
		err = d.refreshNFVMetrics()
		if err != nil {
			return
		}
		d.calcResultMetrics()
	}
	return d.resultMetrics, nil
}

func (d *MetricsDaemon) IsRunning() bool {
	return d.isRunning
}

func (d *MetricsDaemon) Stop() error {
	d.dropNFVMetrics()
	d.dropHostMetrics()
	d.isRunning = false
	d.ticker.Stop()
	return nil
}

func (d *MetricsDaemon) dropNFVMetrics() {
	d.nfvMetrics = []types.Metrics{}
}

func (d *MetricsDaemon) dropHostMetrics() {
	d.hostMetrics = []types.Metrics{}
}
