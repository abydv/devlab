package utils

import (
	"net"
	"strconv"
	"testing"
)

func TestFreePortIsUsable(t *testing.T) {
	port, err := FreePort()
	if err != nil {
		t.Fatalf("FreePort() error = %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("FreePort() = %d, want a valid port number", port)
	}

	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("could not listen on FreePort() result %d: %v", port, err)
	}
	defer l.Close()
}

func TestFreePortReturnsDistinctPorts(t *testing.T) {
	a, err := FreePort()
	if err != nil {
		t.Fatalf("FreePort() error = %v", err)
	}
	b, err := FreePort()
	if err != nil {
		t.Fatalf("FreePort() error = %v", err)
	}
	if a == b {
		t.Errorf("FreePort() returned the same port twice: %d", a)
	}
}
