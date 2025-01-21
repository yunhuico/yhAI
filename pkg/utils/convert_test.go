package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConversion(t *testing.T) {
	type A struct {
		Enabled   bool      `json:"enabled"`
		Progress  time.Time `json:"progress,omitempty"`
		Successes int       `json:"successes,omitempty"`
		Score     float64   `json:"score,omitempty"`
	}

	assert := require.New(t)
	a := A{
		Enabled:   true,
		Progress:  time.UnixMilli(1673504259774).UTC(),
		Successes: 6,
		Score:     9.99,
	}

	m, err := ConvertStructToMap(a)
	assert.NoError(err)

	want := map[string]any{
		"enabled":   true,
		"progress":  "2023-01-12T06:17:39.774Z",
		"successes": float64(6),
		"score":     9.99,
	}
	assert.Equal(want, m)

	var b A
	err = ConvertMapToStruct(want, &b)
	assert.NoError(err)

	assert.Equal(a, b)
}
