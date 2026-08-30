package timecontrol

import (
	"testing"
	"time"
)

func TestForceBlockWins(t *testing.T) {
	now := time.Date(2026, 8, 24, 19, 30, 0, 0, time.UTC)
	state := Evaluate(now, Policy{Enabled: true, Schedule: []ScheduleWindow{{DayOfWeek: 1, StartMinutes: 19 * 60, EndMinutes: 21 * 60}}}, Override{ForceBlocked: true})
	if state.State != StateBlocked || state.Reason != "force_block" {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestExtraTimeExpires(t *testing.T) {
	now := time.Date(2026, 8, 24, 22, 0, 0, 0, time.UTC)
	expires := now.Add(-time.Minute)
	state := Evaluate(now, Policy{Enabled: true}, Override{ExtendedUntil: &expires})
	if state.State != StateBlocked {
		t.Fatalf("expected blocked after extra time expires: %+v", state)
	}
}

func TestScheduleCrossesMidnight(t *testing.T) {
	location := time.FixedZone("ICT", 7*60*60)
	state := Evaluate(time.Date(2026, 8, 24, 0, 30, 0, 0, location), Policy{Enabled: true, Schedule: []ScheduleWindow{{DayOfWeek: 0, StartMinutes: 23 * 60, EndMinutes: 60}}}, Override{})
	if state.State != StateAllowed {
		t.Fatalf("expected allowed: %+v", state)
	}
}

func TestDisabledPolicyAllowsAccess(t *testing.T) {
	state := Evaluate(time.Date(2026, 8, 25, 0, 30, 0, 0, time.UTC), Policy{Enabled: false}, Override{})
	if state.State != StateAllowed || state.Reason != "policy_disabled" {
		t.Fatalf("disabled policy must not lock a machine: %+v", state)
	}
}

func TestScheduleUsesPolicyTimezone(t *testing.T) {
	utcTime := time.Date(2026, 8, 24, 12, 30, 0, 0, time.UTC) // 19:30 in Asia/Ho_Chi_Minh
	state := Evaluate(utcTime, Policy{
		Enabled:  true,
		Timezone: "Asia/Ho_Chi_Minh",
		Schedule: []ScheduleWindow{{DayOfWeek: 1, StartMinutes: 19 * 60, EndMinutes: 20 * 60}},
	}, Override{})
	if state.State != StateAllowed || state.Reason != "schedule" {
		t.Fatalf("expected schedule to use policy timezone: %+v", state)
	}
}

func TestInvalidPolicyTimezoneDoesNotFallBackToUTC(t *testing.T) {
	state := Evaluate(time.Now().UTC(), Policy{
		Enabled:  true,
		Timezone: "Not/A_Real_Timezone",
		Schedule: []ScheduleWindow{{DayOfWeek: 1, StartMinutes: 19 * 60, EndMinutes: 20 * 60}},
	}, Override{})
	if state.State != StateBlocked || state.Reason != "invalid_timezone" {
		t.Fatalf("invalid timezone must not silently evaluate in UTC: %+v", state)
	}
}
