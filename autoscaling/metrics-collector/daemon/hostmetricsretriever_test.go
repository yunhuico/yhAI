package daemon

import (
	"testing"

	"github.com/bmizerany/assert"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

func TestConvert2IP(t *testing.T) {
	var cases = []struct {
		HostPort string
		ExpectIP string
	}{
		{"", ""},
		{"192.168.10.213:10000", "192.168.10.213"},
		{"192.168.10.213", "192.168.10.213"},
		{"some.cadvisor.addr", "some.cadvisor.addr"},
	}
	for _, c := range cases {
		gotIP := convert2IP(c.HostPort)
		assert.Equal(t, c.ExpectIP, gotIP)
	}
}

func TestComposeHostMetricsLines(t *testing.T) {
	var lines1 []types.Line
	lines1 = append(lines1, types.Line{
		Index: "host_cpu_usage",
		Map: map[string]string{
			"alert":   "true",
			"host_ip": "5.6.7.8",
		},
		FloatVar: 1.2,
	})
	lines1 = append(lines1, types.Line{
		Index: "host_memory_usage",
		Map: map[string]string{
			"alert":   "true",
			"host_ip": "5.6.7.8",
		},
		FloatVar: 3.4,
	})
	var cases = []struct {
		CPUUsage    float64
		MemUsage    float64
		HostIP      string
		ExpectLines []types.Line
	}{
		{0, 0, "", nil},
		{1.2, 3.4, "5.6.7.8", lines1},
	}
	for _, c := range cases {
		gotLines := composeHostMetricsLines(c.CPUUsage, c.MemUsage, c.HostIP)
		// fmt.Printf("got lines: %+v\n", gotLines)
		assert.Equal(t, c.ExpectLines, gotLines)
	}
}
