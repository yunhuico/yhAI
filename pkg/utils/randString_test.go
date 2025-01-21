package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	set2 "jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"
)

func TestRandStr(t *testing.T) {
	str, err := RandStr(32)
	assert.NoError(t, err)
	assert.Len(t, str, 32)
	str2, err := RandStr(32)
	assert.NoError(t, err)
	assert.NotEqual(t, str, str2)

	set := set2.Set[string]{}
	for i := 0; i < 3200; i++ {
		str, err := RandStr(32)
		assert.NoError(t, err)
		set.Add(str)
	}
	assert.Len(t, set, 3200)
}
