package utils

import "testing"

func TestNewIDIsUniqueAndWellFormed(t *testing.T) {
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id, err := NewID()
		if err != nil {
			t.Fatalf("NewID() error = %v", err)
		}
		if len(id) != 12 {
			t.Fatalf("NewID() length = %d, want 12", len(id))
		}
		if seen[id] {
			t.Fatalf("NewID() produced duplicate id %q", id)
		}
		seen[id] = true
	}
}
