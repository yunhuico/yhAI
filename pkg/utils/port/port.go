package port

import (
	"fmt"
	"net"
	"time"

	"github.com/phayes/freeport"
)

// GetFreePort returns a free port
func GetFreePort() (int, error) {
	return freeport.GetFreePort()
}

// WaitPort waits for a port to be available
func WaitPort(port int, duration time.Duration) error {
	conn, _ := net.DialTimeout("tcp", net.JoinHostPort("localhost", fmt.Sprintf("%d", port)), duration)

	if conn == nil {
		return fmt.Errorf("port %d is not available", port)
	}

	return conn.Close()
}
