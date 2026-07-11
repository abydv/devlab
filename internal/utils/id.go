// Package utils holds small helpers shared across internal packages.
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewID returns a random, URL-safe, unique identifier suitable for use
// as a Workspace ID or other on-disk resource name.
func NewID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("utils: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
