package timecontrol

import (
	"testing"
	"time"
)

func TestIsScheduledAllowedAcrossMidnight(t *testing.T) {
	location := time.FixedZone("ICT", 7*60*60)
	windows := []ScheduleWindow{{DayOfWeek: 1, StartMinutes: 23 * 60, EndMinutes: 60}}

	if !IsScheduledAllowed(time.Date(2026, 8, 24, 23, 30, 0, 0, location), windows) { // Monday
		t.Fatal("expected late Monday to be allowed")
	}
	if !IsScheduledAllowed(time.Date(2026, 8, 25, 0, 30, 0, 0, location), windows) { // Tuesday
		t.Fatal("expected early Tuesday to be allowed by Monday window")
	}
	if IsScheduledAllowed(time.Date(2026, 8, 25, 1, 0, 0, 0, location), windows) {
		t.Fatal("expected schedule endpoint to be exclusive")
	}
}

func TestNextAllowedAt(t *testing.T) {
	location := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 8, 24, 8, 0, 0, 0, location) // Monday
	next := NextAllowedAt(now, []ScheduleWindow{{DayOfWeek: 1, StartMinutes: 9 * 60, EndMinutes: 10 * 60}})
	if next == nil || next.Hour() != 9 || next.Minute() != 0 {
		t.Fatalf("unexpected next time: %v", next)
	}
}
