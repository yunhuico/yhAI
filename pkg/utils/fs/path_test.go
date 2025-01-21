package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAbsPath(t *testing.T) {
	absPath := "/tmp/.ultrafox/dist"
	path, err := GetAbsPath(absPath)
	assert.NoError(t, err)
	assert.Equal(t, absPath, path)

	homePath := "~/.ultrafox/dist"
	_, err = GetAbsPath(homePath)
	assert.NoError(t, err)
}
