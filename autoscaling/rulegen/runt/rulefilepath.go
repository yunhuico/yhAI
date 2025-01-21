package runt

import (
	"os"
	"path/filepath"

	"linkernetworks.com/dcos-backend/autoscaling/rulegen/consts"
)

var (
	// RuleFilePath is the final path of Prometheus rules.
	// It's initialized in funciton init()
	RuleFilePath string
)

// InitRuleFilePath get rule file path from env and check if the directory exists.
func InitRuleFilePath() error {
	// use rule file path from env, or default if not set
	RuleFilePath = consts.DefaultRuleFile
	if r := os.Getenv(consts.EnvRuleFile); len(r) > 0 {
		RuleFilePath = r
	}
	// check file directory
	if _, err := os.Stat(filepath.Dir(RuleFilePath)); err != nil {
		return err
	}
	return nil
}
