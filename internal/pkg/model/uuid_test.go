package model

import "testing"

func TestOptionalUUID(t *testing.T) {
	for _, s := range []string{"", "  ", "CUS-0001", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		if got := OptionalUUID(s); got != nil {
			t.Errorf("OptionalUUID(%q) = %q, want nil", s, *got)
		}
	}
	got := OptionalUUID(" D1F5E310-365D-5E33-930D-AD82B4F1B53C ")
	if got == nil || *got != "d1f5e310-365d-5e33-930d-ad82b4f1b53c" {
		t.Errorf("valid uuid not normalised: %v", got)
	}
}
