package daily

import "time"

func ComputeNextStreak(previous Streak, currentDate time.Time, graceHours int) Streak {
	return ComputeNextStreakWithFreeze(previous, currentDate, graceHours, 0, false)
}

// ComputeNextStreakWithFreeze computes the next streak. If useFreeze is true and diff > grace
// but within grace+24h, the streak continues (freeze consumed).
func ComputeNextStreakWithFreeze(previous Streak, currentDate time.Time, graceHours int, freezeAvailable int, useFreeze bool) Streak {
	updated := previous
	if updated.LastDate == nil {
		updated.Current = 1
		updated.Longest = max(updated.Longest, updated.Current)
		updated.LastDate = ptrTime(currentDate)
		return updated
	}

	diff := currentDate.Sub(updated.LastDate.UTC())
	if diff < 0 {
		return previous
	}
	graceDur := time.Duration(graceHours) * time.Hour
	gracePlus24h := graceDur + 24*time.Hour
	if diff <= graceDur {
		updated.Current++
	} else if useFreeze && freezeAvailable > 0 && diff <= gracePlus24h {
		updated.Current++
		updated.FreezeCount++
	} else {
		updated.Current = 1
	}
	if updated.Current > updated.Longest {
		updated.Longest = updated.Current
	}
	updated.LastDate = ptrTime(currentDate)
	return updated
}

func ptrTime(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
