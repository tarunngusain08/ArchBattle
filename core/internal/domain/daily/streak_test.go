package daily

import (
	"testing"
	"time"
)

var graceHours = 48

func ptrT(t time.Time) *time.Time { return &t }

func TestComputeNextStreak(t *testing.T) {
	base := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	t.Run("first submission", func(t *testing.T) {
		streak := ComputeNextStreak(Streak{}, base, graceHours)
		if streak.Current != 1 || streak.Longest != 1 {
			t.Errorf("first submission: got current=%d longest=%d", streak.Current, streak.Longest)
		}
	})

	t.Run("consecutive day extends streak", func(t *testing.T) {
		prev := Streak{Current: 3, Longest: 3, LastDate: ptrT(base)}
		next := ComputeNextStreak(prev, base.Add(24*time.Hour), graceHours)
		if next.Current != 4 || next.Longest != 4 {
			t.Errorf("consecutive day: got current=%d longest=%d", next.Current, next.Longest)
		}
	})

	t.Run("within grace period extends streak", func(t *testing.T) {
		prev := Streak{Current: 2, Longest: 2, LastDate: ptrT(base)}
		next := ComputeNextStreak(prev, base.Add(47*time.Hour), graceHours)
		if next.Current != 3 {
			t.Errorf("grace period: expected current=3, got %d", next.Current)
		}
	})

	t.Run("beyond grace resets streak", func(t *testing.T) {
		prev := Streak{Current: 5, Longest: 5, LastDate: ptrT(base)}
		next := ComputeNextStreak(prev, base.Add(72*time.Hour), graceHours)
		if next.Current != 1 {
			t.Errorf("reset: expected current=1, got %d", next.Current)
		}
		if next.Longest != 5 {
			t.Errorf("reset: longest should be preserved at 5, got %d", next.Longest)
		}
	})
}
