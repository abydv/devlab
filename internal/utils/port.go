package utils

import (
	"fmt"
	"net"
)

// FreePort returns a currently unused TCP port on the host, suitable
// for publishing a container port without colliding with another
// workspace's service.
func FreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("utils: find free port: %w", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
