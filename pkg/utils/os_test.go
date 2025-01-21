package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHostID(t *testing.T) {
	hostID := GetOSUsername()
	assert.Equal(t, GetOSUsername(), hostID)
	assert.False(t, strings.Contains(hostID, "."))
}

func TestGetUltrafoxHome(t *testing.T) {
	ultrafoxHome := GetUltrafoxHome()
	assert.NotEmpty(t, ultrafoxHome)
	t.Logf("default ultrafox home: %s", ultrafoxHome)
}
