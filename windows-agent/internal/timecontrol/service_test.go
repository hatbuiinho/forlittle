package timecontrol

import (
	"testing"
	"time"
)

func TestSameEffectiveStateComparesTimesByValue(t *testing.T) {
	first := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	second := first
	left := EffectiveState{State: StateBlocked, Reason: "outside_schedule", NextAllowedAt: &first}
	right := EffectiveState{State: StateBlocked, Reason: "outside_schedule", NextAllowedAt: &second}
	if !sameEffectiveState(left, right) {
		t.Fatal("equal timestamps should not produce repeated state changes")
	}
}
