package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdapterClassSpecClass(t *testing.T) {
	var a AdapterClass = "ultrafox/debug"
	assert.Equal(t, "ultrafox/debug#print", a.SpecClass("print"))
}
