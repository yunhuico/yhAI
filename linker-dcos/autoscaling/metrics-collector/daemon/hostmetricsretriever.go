package daemon

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	cadvisor "github.com/google/cadvisor/client"
	"github.com/google/cadvisor/info/v1"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/constant"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

type HostMetricsRetriever struct {
	cadvisorClient *cadvisor.Client
	Config         HostMetricsRetrieverConfig
}

type HostMetricsRetrieverConfig struct {
	// Cadvisor is endpoint of cAdvisor, e.g. "<IP>:<Port>"
	Cadvisor string
	// optional, using default if not set, in Millisecond
	CadvisorTimeout int
}

func NewHostMetricsRetriever(config HostMetricsRetrieverConfig) (retriever *HostMetricsRetriever,
	err error) {
	retriever = &HostMetricsRetriever{}
	if config.CadvisorTimeout <= 0 {
		config.CadvisorTimeout = defaultCadvisorTimeout
	}
	retriever.Config = config
	c, err := cadvisor.NewClient(fmt.Sprintf("http://%s/", config.Cadvisor))
	if err != nil {
		return nil, err
	}
	retriever.cadvisorClient = c
	return
}

// RetrieveMetrics calls cAdvisor API and retrieves all metrics
func (self *HostMetricsRetriever) Retrieve(wg *sync.WaitGroup,
	hostMetricsCh chan *types.Metrics, errCh chan *error) {
	// fmt.Println("host retrieve func called")
	defer wg.Done()

	minfo, err := self.cadvisorClient.MachineInfo()
	if err != nil {
		errCh <- &err
		return
	}
	// fmt.Printf("num cores: %d\n", minfo.NumCores)
	cinfo, err := self.cadvisorClient.ContainerInfo("", &v1.ContainerInfoRequest{NumStats: 2})
	if err != nil {
		errCh <- &err
		return
	}
	if !cinfo.Spec.HasCpu || !cinfo.Spec.HasMemory {
		err = errors.New("do not have CPU or memory")
		errCh <- &err
		return
	}
	if len(cinfo.Stats) < 2 {
		err = errors.New("stats length less than 2")
		errCh <- &err
		return
	}
	// CPU, memory calculation is inspired by cAdvisor portal
	// https://github.com/google/cadvisor/blob/53820123e6167f8aa84f2201920d823369a88911/pages/assets/js/containers.js#L358
	var cpuUsage, memUsage float64
	cur := cinfo.Stats[len(cinfo.Stats)-1]
	prev := cinfo.Stats[len(cinfo.Stats)-2]
	cpuRawUsage := float64(cur.Cpu.Usage.Total - prev.Cpu.Usage.Total)
	intervalNs := float64(cur.Timestamp.Sub(prev.Timestamp).Nanoseconds())
	// fmt.Printf("cpuRawUsage: %f, intervalNs: %f\n", cpuRawUsage, intervalNs)
	cpuUsage = ((cpuRawUsage / intervalNs) / float64(minfo.NumCores)) * 100
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	limit := cinfo.Spec.Memory.Limit
	if limit > minfo.MemoryCapacity {
		limit = minfo.MemoryCapacity
	}
	memUsage = (float64(cur.Memory.Usage) / float64(limit)) * 100

	hostIP := convert2IP(self.Config.Cadvisor)
	// print msg to screen
	log.Printf("Host: %s, CPU: %0.2f%%, Mem: %.2f%%\n", hostIP, cpuUsage, memUsage)
	lines := composeHostMetricsLines(cpuUsage, memUsage, hostIP)
	metrics := types.Metrics{
		Lines: lines,
	}
	hostMetricsCh <- &metrics
	return
}

func composeHostMetricsLines(cpuUsage, memUsage float64, hostIP string) (lines []types.Line) {
	if hostIP == "" {
		return
	}
	lines = append(lines, types.Line{
		Index: constant.IndexHostCPUUsage,
		Map: map[string]string{
			constant.KeyAlert:  "true",
			constant.KeyHostIP: hostIP,
		},
		FloatVar: cpuUsage,
	})
	lines = append(lines, types.Line{
		Index: constant.IndexHostMemUsage,
		Map: map[string]string{
			constant.KeyAlert:  "true",
			constant.KeyHostIP: hostIP,
		},
		FloatVar: memUsage,
	})
	return
}

// convert <IP>:<Port> to <IP>
func convert2IP(hostPort string) string {
	return strings.Split(hostPort, ":")[0]
}

// retrieve multiple host metrics
func retrieveMultipleHostMetrics(cadvisors []string, timeout int) (metricsArr []types.Metrics, err error) {
	// 1. Retrieve metrics from all cAdvisors asynchronously(in each goroutine) and
	//    These lines are ones related with host machines, they are CPU and memory usage.
	//    If request to cAdvisor timeout, ignore.
	var nRoutines = len(cadvisors)
	// log.Printf("starting %d routines to retrieve metrics\n", nRoutines)
	var wg sync.WaitGroup
	wg.Add(nRoutines)
	var metricsCh = make(chan *types.Metrics, nRoutines)
	var errCh = make(chan *error, nRoutines)

	for _, address := range cadvisors {
		config := HostMetricsRetrieverConfig{
			Cadvisor:        address,
			CadvisorTimeout: timeout,
		}
		retriever, err := NewHostMetricsRetriever(config)
		if err != nil {
			log.Printf("new metrics retriever error: %v\n", err)
			continue
		}
		go retriever.Retrieve(&wg, metricsCh, errCh)
	}

	// 2. Wait for all goroutine to finish, and gather all filtered metrics
	//   into one array.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < nRoutines; i++ {
			select {
			case r, ok := <-metricsCh:
				if ok {
					// log.Printf("len of metrics lines: %d\n", len(r.Lines))
					metricsArr = append(metricsArr, *r)
				}
			case e, ok := <-errCh:
				if ok {
					log.Printf("retrieve cadvisor metrics error: %v\n", *e)
				}
			}
		}
	}()
	wg.Wait()
	return
}
