package svc

import (
	"os"
)

// overwriteRuleFile clears file and write content to it
func overwriteRuleFile(content []byte, filePath string) error {
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	// https://joeshaw.org/dont-defer-close-on-writable-files/
	// defer f.Close()
	// clear file content
	if err = f.Truncate(0); err != nil {
		return err
	}
	if _, err = f.Write(content); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}
