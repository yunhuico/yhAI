package tunnel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrpClint(t *testing.T) {
	client := NewTunnelClient("ultrafox", "uktrafox.io:7500", "token")
	defer client.Close()
	assert.Error(t, client.Start(context.Background()))
	client.RegisterHTTP(8010, "api")
	assert.NoError(t, client.init())
}
