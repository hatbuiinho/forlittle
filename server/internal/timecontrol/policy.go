package timecontrol

import (
	"fmt"
	"sort"
	"time"
)

const (
	StateAllowed  = "ALLOWED"
	StateBlocked  = "BLOCKED"
	StateExtended = "EXTENDED"
)

type ScheduleWindow struct {
	DayOfWeek    int `json:"day_of_week"`
	StartMinutes int `json:"start_minutes"`
	EndMinutes   int `json:"end_minutes"`
}

func ValidateSchedule(windows []ScheduleWindow) error {
	for _, window := range windows {
		if window.DayOfWeek < 0 || window.DayOfWeek > 6 {
			return fmt.Errorf("day_of_week must be between 0 and 6")
		}
		if window.StartMinutes < 0 || window.StartMinutes >= 24*60 || window.EndMinutes < 0 || window.EndMinutes >= 24*60 {
			return fmt.Errorf("schedule minutes must be between 0 and 1439")
		}
		if window.StartMinutes == window.EndMinutes {
			return fmt.Errorf("schedule window cannot have equal start and end")
		}
	}

	return nil
}

// IsScheduledAllowed evaluates the current day and the previous day's overnight
// windows. Go's Weekday uses Sunday=0, which is also the API representation.
func IsScheduledAllowed(now time.Time, windows []ScheduleWindow) bool {
	minute := now.Hour()*60 + now.Minute()
	today := int(now.Weekday())
	previousDay := (today + 6) % 7

	for _, window := range windows {
		if window.StartMinutes < window.EndMinutes && window.DayOfWeek == today && minute >= window.StartMinutes && minute < window.EndMinutes {
			return true
		}

		if window.StartMinutes > window.EndMinutes {
			if window.DayOfWeek == today && minute >= window.StartMinutes {
				return true
			}
			if window.DayOfWeek == previousDay && minute < window.EndMinutes {
				return true
			}
		}
	}

	return false
}

func NextAllowedAt(now time.Time, windows []ScheduleWindow) *time.Time {
	if len(windows) == 0 {
		return nil
	}

	candidates := make([]time.Time, 0, len(windows)*2)
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
