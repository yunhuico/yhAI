package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNowDurationFrom(t *testing.T) {
	duration := NowHumanDurationFrom(time.Now().Add(time.Second * -10))
	assert.Equal(t, 10*time.Second, duration)
}
