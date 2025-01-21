package constant

const (
	// IndexCPUHigh is index from cAdvisor
	IndexCPUHigh = "container_cpu_usage_high_result"
	// IndexCPULow is index from cAdvisor
	IndexCPULow = "container_cpu_usage_low_result"
	// IndexMemHigh is index from cAdvisor
	IndexMemHigh = "container_memory_usage_high_result"
	// IndexMemLow is index from cAdvisor
	IndexMemLow = "container_memory_usage_low_result"
)

const (
	// IndexCPUAvgHigh is index for Prometheus
	IndexCPUAvgHigh = "container_cpu_usage_high_result"
	// IndexCPUAvgLow is index for Prometheus
	IndexCPUAvgLow = "container_cpu_usage_low_result"
	// IndexMemAvgHigh is index for Prometheus
	IndexMemAvgHigh = "container_memory_usage_high_result"
	// IndexMemAvgLow is index for Prometheus
	IndexMemAvgLow = "container_memory_usage_low_result"
)

const (
	// IndexHostCPUUsage is index for Prometheus
	IndexHostCPUUsage = "host_cpu_usage"
	// IndexHostMemUsage is index for Prometheus
	IndexHostMemUsage = "host_memory_usage"
)
