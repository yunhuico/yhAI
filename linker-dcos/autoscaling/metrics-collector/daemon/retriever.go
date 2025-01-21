package daemon

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

const (
	// consider timeout after 5000ms by default when GET cadvisor/metrics
	defaultCadvisorTimeout = 5000
)

var (
	// ErrInvalidAddress is returned when cadvisor endpoint is not valid
	ErrInvalidAddress = errors.New("invalid param address")
	ErrInvalidTimeout = errors.New("invalid param timeout")
)

type MetricsRetriever struct {
	Config MetricsRetrieverConfig
}

type MetricsRetrieverConfig struct {
	// Cadvisor is endpoint of cAdvisor, e.g. "<IP>:<Port>"
	Cadvisor string
	// optional, using default if not set, in Millisecond
	CadvisorTimeout int
}

func NewMetricsRetriever(config MetricsRetrieverConfig) (retriever *MetricsRetriever,
	err error) {
	retriever = &MetricsRetriever{}
	if config.CadvisorTimeout <= 0 {
		config.CadvisorTimeout = defaultCadvisorTimeout
	}
	retriever.Config = config
	return
}

// RetrieveMetrics calls cAdvisor API and retrieves all metrics
func (r *MetricsRetriever) RetrieveMetrics(wg *sync.WaitGroup,
	cadvisorMetricsCh chan *types.RawMetrics, errCh chan *error) {

	defer wg.Done()

	cadvisorMetrics := &types.RawMetrics{}
	resp, err := getCadvisorMetrics(r.Config.Cadvisor, r.Config.CadvisorTimeout)
	if err != nil {
		errCh <- &err
		return
	}
	defer resp.Body.Close()

	var lines []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	cadvisorMetrics.Lines = lines
	cadvisorMetricsCh <- cadvisorMetrics
	return
}

// RetrieveMetrics calls cAdvisor API and retrieves filtered metrics
func (r *MetricsRetriever) RetrieveMetricsWithFilter(f Filter,
	wg *sync.WaitGroup, cadvisorMetricsCh chan *types.RawMetrics, errCh chan *error) {

	defer wg.Done()

	cadvisorMetrics := &types.RawMetrics{}
	resp, err := getCadvisorMetrics(r.Config.Cadvisor, r.Config.CadvisorTimeout)
	if err != nil {
		errCh <- &err
		return
	}
	defer resp.Body.Close()

	var lines []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		// filter
		useful := f.Filter(line)
		if useful {
			lines = append(lines, line)
		}
	}

	cadvisorMetrics.Lines = lines
	cadvisorMetricsCh <- cadvisorMetrics
	return
}

// GET <endpoint>/metrics
func getCadvisorMetrics(endpoint string, timeout int) (*http.Response, error) {
	apiURL := fmt.Sprintf("http://%s/metrics", endpoint)
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}
	return client.Get(apiURL)
}

// retrieve multiple filtered metrics
func retrieveMultipleMetrics(cadvisors []string, timeout int, filter Filter) (filteredMetricsArr []types.RawMetrics, err error) {
	// 1. Retrieve metrics from all cAdvisors asynchronously(in each goroutine) and
	//    filter out useful lines(called filteredMetricsCh) from original metrics.
	//    These lines are ones related with NFV containers, it's CPU and memory
	//	  usage of PGW/SGW containers.
	//    If request to cAdvisor timeout, ignore.
	var nRoutines = len(cadvisors)
	// log.Printf("starting %d routines to retrieve metrics\n", nRoutines)
	var wg sync.WaitGroup
	wg.Add(nRoutines)
	var filteredMetricsCh = make(chan *types.RawMetrics, nRoutines)
	var errCh = make(chan *error, nRoutines)

	for _, address := range cadvisors {
		config := MetricsRetrieverConfig{
			Cadvisor:        address,
			CadvisorTimeout: timeout,
		}
		retriever, err := NewMetricsRetriever(config)
		if err != nil {
			log.Printf("new metrics retriever error: %v\n", err)
			continue
		}
		go retriever.RetrieveMetricsWithFilter(filter, &wg, filteredMetricsCh, errCh)
		// go retriever.RetrieveMetrics(&wg, filteredMetricsCh, errCh)
	}

	// 2. Wait for all goroutine to finish, and gather all filtered metrics
	//   into one array.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < nRoutines; i++ {
			select {
			case r, ok := <-filteredMetricsCh:
				if ok {
					// log.Printf("len of metrics lines: %d\n", len(r.Lines))
					filteredMetricsArr = append(filteredMetricsArr, *r)
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
