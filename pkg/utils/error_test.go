package utils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnwrapError(t *testing.T) {
	originalErr := errors.New("mock err")
	var err = originalErr
	for i := 0; i < 100; i++ {
		err = fmt.Errorf("wrapping error: %w", err)
	}
	actualErr := UnwrapError(err)
	assert.Equal(t, actualErr, originalErr)
}
