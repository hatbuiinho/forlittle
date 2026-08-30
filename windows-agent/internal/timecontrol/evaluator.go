package timecontrol

import (
	"sort"
	"time"
)

func Evaluate(now time.Time, policy Policy, override Override) EffectiveState {
	if override.ForceBlocked {
		return EffectiveState{State: StateBlocked, Reason: "force_block"}
	}
	if override.ManualUnlockUntil != nil && now.Before(*override.ManualUnlockUntil) {
		return EffectiveState{State: StateExtended, Reason: "manual_unblock", ExtendedUntil: override.ManualUnlockUntil}
	}
	if override.ExtendedUntil != nil && now.Before(*override.ExtendedUntil) {
		return EffectiveState{State: StateExtended, Reason: "extra_time", ExtendedUntil: override.ExtendedUntil}
	}
	if !policy.Enabled {
		return EffectiveState{State: StateAllowed, Reason: "policy_disabled"}
	}
	if policy.Timezone != "" {
		location, err := time.LoadLocation(policy.Timezone)
		if err != nil {
			// Never fall back to UTC here: it would enforce a valid local schedule at the wrong hour.
			return EffectiveState{State: StateBlocked, Reason: "invalid_timezone"}
		}
		now = now.In(location)
	}
	if isScheduled(now, policy.Schedule) {
		return EffectiveState{State: StateAllowed, Reason: "schedule"}
	}
	return EffectiveState{State: StateBlocked, Reason: "outside_schedule", NextAllowedAt: nextAllowed(now, policy.Schedule)}
}

func isScheduled(now time.Time, windows []ScheduleWindow) bool {
	minute := now.Hour()*60 + now.Minute()
	today := int(now.Weekday())
	previous := (today + 6) % 7
	for _, window := range windows {
		if window.StartMinutes < window.EndMinutes && window.DayOfWeek == today && minute >= window.StartMinutes && minute < window.EndMinutes {
			return true
		}
		if window.StartMinutes > window.EndMinutes {
			if window.DayOfWeek == today && minute >= window.StartMinutes {
				return true
			}
			if window.DayOfWeek == previous && minute < window.EndMinutes {
				return true
			}
		}
	}
	return false
}

func nextAllowed(now time.Time, windows []ScheduleWindow) *time.Time {
	candidates := make([]time.Time, 0, len(windows))
	for offset := 0; offset <= 7; offset++ {
		day := now.AddDate(0, 0, offset)
		for _, window := range windows {
			if int(day.Weekday()) != window.DayOfWeek {
				continue
			}
			candidate := time.Date(day.Year(), day.Month(), day.Day(), window.StartMinutes/60, window.StartMinutes%60, 0, 0, now.Location())
			if candidate.After(now) {
				candidates = append(candidates, candidate)
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Before(candidates[j]) })
	return &candidates[0]
}
