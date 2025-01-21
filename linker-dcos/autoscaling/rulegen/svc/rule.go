package svc

import (
	"log"
)

func UpdateRules(cpuEnabled, memEnabled bool, cpuHigh, cpuLow, memHigh, memLow float32, duration, ruleFile string) error {
	log.Printf("update rules(cpuEnable=%v, memEnabled=%v, cpuHigh=%f, cpuLow=%f, memHigh=%f, memLow=%f, duration=%s\n)",
		cpuEnabled, memEnabled, cpuHigh, cpuLow, memHigh, memLow, duration)

	// generate rules
	content, err := genRuleFile(cpuEnabled, memEnabled, cpuHigh, cpuLow, memHigh, memLow, duration)
	if err != nil {
		log.Printf("marshal rules to bytes error: %v\n", err)
		return err
	}
	// save to file
	if err := overwriteRuleFile(content, ruleFile); err != nil {
		log.Printf("save rules to file error: %v\n", err)
		return err
	}
	return nil
}
