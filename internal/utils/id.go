// Package utils holds small helpers shared across internal packages.
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewID returns a random, URL-safe, unique identifier suitable for use
// as a Workspace ID or other on-disk resource name.
//
// Its length (6 bytes, 12 hex characters) is deliberately short enough
// that "devlab-<id>-kubernetes" fits within k3d's 32-character cluster
// name limit, DevLab's tightest naming constraint — see
// internal/service/factory. 48 bits of randomness is far more than
// sufficient for a personal, single-user tool's workspace count.
func NewID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("utils: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
