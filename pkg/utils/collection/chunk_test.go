package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunk(t *testing.T) {
	assert.Nil(t, Chunk([]string{}, 100))

	input := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	group := Chunk(input, 3)
	assert.Len(t, group, 9)
	assert.Equal(t, 9, cap(group))
}
