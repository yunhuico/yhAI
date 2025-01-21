package svc

import (
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/consts"
	"linkernetworks.com/dcos-backend/autoscaling/rulegen/rule"
)

func genRuleFile(cpuEnabled, memEnabled bool, cpuHigh, cpuLow, memHigh, memLow float32, duration string) (fileContent []byte, err error) {
	var rules rule.PromRules
	if cpuEnabled {
		rules = append(rules, cpuHigh2PromRule(cpuHigh, duration))
		rules = append(rules, cpuLow2PromRule(cpuLow, duration))
	}
	if memEnabled {
		rules = append(rules, memHigh2PromRule(memHigh, duration))
		rules = append(rules, memLow2PromRule(memLow, duration))
	}
	return rules.Marshal()
}

func cpuHigh2PromRule(cpuHigh float32, duration string) rule.PromRule {
	return rule.PromRule{
		AlertName:  consts.HostHighCPUAlert,
		Conditions: *rule.NewConditions(rule.Condition{Index: consts.IndexHostCPUUsage, CompareSym: ">", Threshold: cpuHigh}),
		Duration:   duration,
		Annotations: map[string]string{
			"summary":     "High CPU usage alert for host machine",
			"description": "High CPU usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
		},
	}
}

func cpuLow2PromRule(cpuLow float32, duration string) rule.PromRule {
	return rule.PromRule{
		AlertName:  consts.HostLowCPUAlert,
		Conditions: *rule.NewConditions(rule.Condition{Index: consts.IndexHostCPUUsage, CompareSym: "<", Threshold: cpuLow}),
		Duration:   duration,
		Annotations: map[string]string{
			"summary":     "Low CPU usage alert for host machine",
			"description": "Low CPU usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
		},
	}
}

func memHigh2PromRule(memHigh float32, duration string) rule.PromRule {
	return rule.PromRule{
		AlertName:  consts.HostHighMemoryAlert,
		Conditions: *rule.NewConditions(rule.Condition{Index: consts.IndexHostMemUsage, CompareSym: ">", Threshold: memHigh}),
		Duration:   duration,
		Annotations: map[string]string{
			"summary":     "High memory usage alert for host machine",
			"description": "High memory usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
		},
	}
}

func memLow2PromRule(memLow float32, duration string) rule.PromRule {
	return rule.PromRule{
		AlertName:  consts.HostLowMemoryAlert,
		Conditions: *rule.NewConditions(rule.Condition{Index: consts.IndexHostMemUsage, CompareSym: "<", Threshold: memLow}),
		Duration:   duration,
		Annotations: map[string]string{
			"summary":     "Low memory usage alert for host machine",
			"description": "Low memory usage for host machine on {{$labels.host_ip}}, (current value: {{$value}})",
		},
	}
}
