package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShortNanoID(t *testing.T) {
	id, err := ShortNanoID()
	require.NoError(t, err)
	require.Len(t, id, 9)
}

func TestNanoID(t *testing.T) {
	id, err := NanoID()
	require.NoError(t, err)
	require.Len(t, id, 16)
}

func TestLongNanoID(t *testing.T) {
	id, err := LongNanoID()
	require.NoError(t, err)
	require.Len(t, id, 21)
}
