package match

import (
	"math"
	"sort"
)

func BuildOrderingScore(serverTS int64, seq int64) float64 {
	return float64(serverTS) + float64(seq)/1e9
}

func IsCorrectAnswer(actual []int, expected []int) bool {
	if len(actual) != len(expected) {
		return false
	}

	lhs := append([]int(nil), actual...)
	rhs := append([]int(nil), expected...)
	sort.Ints(lhs)
	sort.Ints(rhs)
	for idx := range lhs {
		if lhs[idx] != rhs[idx] {
			return false
		}
	}
	return true
}

// SpeedMultiplier returns a score bonus based on how quickly the player answered
// relative to others. rank is 0-indexed (0 = first to answer), total is the
// number of players who have answered including this player (totalAfter).
// First answerer: rank=0, total=1 → percentile=0.0 → 1.5x bonus.
func SpeedMultiplier(rank int64, total int64) float64 {
	if total <= 0 {
		return 1.0
	}
	// percentile is 0.0 for the first answerer, approaching 1.0 for the last.
	percentile := float64(rank) / float64(total)
	switch {
	case percentile < 0.25:
		return 1.5
	case percentile < 0.50:
		return 1.2
	default:
		return 1.0
	}
}

func CalculatePoints(rank int64, total int64, correct bool, elapsedSeconds int) int {
	if !correct {
		return 0
	}
	points := 100.0 * SpeedMultiplier(rank, total)
	if elapsedSeconds > 45 {
		points *= 1.1
	}
	return int(math.Round(points))
}
