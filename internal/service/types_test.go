package service

import "testing"

func TestIsKnownType(t *testing.T) {
	for _, typ := range KnownTypes {
		if !IsKnownType(typ) {
			t.Errorf("IsKnownType(%q) = false, want true", typ)
		}
	}

	if IsKnownType("nonsense") {
		t.Error(`IsKnownType("nonsense") = true, want false`)
	}
	if IsKnownType("") {
		t.Error(`IsKnownType("") = true, want false`)
	}
}
