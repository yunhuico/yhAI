package port

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetFreePort(t *testing.T) {
	port1, err := GetFreePort()
	assert.NoError(t, err)
	port2, err := GetFreePort()
	assert.NoError(t, err)

	assert.NotEqual(t, port1, port2)

	err = WaitPort(port1, 100*time.Millisecond)
	assert.Error(t, err)
	t.Logf("error is %v", err)

	go func() {
		_ = http.ListenAndServe(fmt.Sprint(":", port1), nil)
	}()

	_ = WaitPort(port1, 1000*time.Millisecond)
}
