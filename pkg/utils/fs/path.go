package fs

import (
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

// GetAbsPath returns the absolute path of the given path.
func GetAbsPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	return homedir.Expand(path)
}
