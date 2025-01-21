package types

type ReqPutRules struct {
	CPUEnabled bool `json:"cpu_enabled"`
	MemEnabled bool `json:"mem_enabled"`
	// Duration: unix time duration like "30s", "5m"
	Duration string `json:"duration"`
	// Thresholds: cpu_high, cpu_low, mem_high, mem_low
	Thresholds map[string]float32 `json:"thresholds"`
}
