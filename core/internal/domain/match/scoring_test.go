package match

import "testing"

func TestSpeedMultiplier(t *testing.T) {
	tests := []struct {
		name  string
		rank  int64
		total int64
		want  float64
	}{
		// First answerer: rank=0, total=1 → percentile=0.0 → fastest bucket
		{name: "first of one", rank: 0, total: 1, want: 1.5},
		// First of four: rank=0, total=4 → percentile=0.0 → fastest
		{name: "first of four", rank: 0, total: 4, want: 1.5},
		// Second of four: rank=1, total=4 → percentile=0.25 → not fastest (>=0.25)
		{name: "second of four", rank: 1, total: 4, want: 1.2},
		// Third of four: rank=2, total=4 → percentile=0.5 → default (>=0.50)
		{name: "third of four", rank: 2, total: 4, want: 1.0},
		// Fourth of four: rank=3, total=4 → percentile=0.75 → default
		{name: "last of four", rank: 3, total: 4, want: 1.0},
		// Edge: total=0 guard
		{name: "total zero", rank: 0, total: 0, want: 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SpeedMultiplier(tc.rank, tc.total)
			if got != tc.want {
				t.Errorf("SpeedMultiplier(%d, %d) = %.1f, want %.1f", tc.rank, tc.total, got, tc.want)
			}
		})
	}
}

func TestCalculatePoints(t *testing.T) {
	tests := []struct {
		name    string
		rank    int64
		total   int64
		correct bool
		elapsed int
		wantMin int
		wantMax int
	}{
		{name: "wrong answer", rank: 0, total: 1, correct: false, elapsed: 10, wantMin: 0, wantMax: 0},
		{name: "fastest correct", rank: 0, total: 1, correct: true, elapsed: 10, wantMin: 150, wantMax: 150},
		{name: "fastest with slow penalty", rank: 0, total: 1, correct: true, elapsed: 50, wantMin: 165, wantMax: 165},
		{name: "last correct", rank: 3, total: 4, correct: true, elapsed: 10, wantMin: 100, wantMax: 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculatePoints(tc.rank, tc.total, tc.correct, tc.elapsed)
			if got < tc.wantMin || got > tc.wantMax {
				t.Errorf("CalculatePoints(%d,%d,%v,%d) = %d, want [%d, %d]", tc.rank, tc.total, tc.correct, tc.elapsed, got, tc.wantMin, tc.wantMax)
			}
		})
	}
}

func TestIsCorrectAnswer(t *testing.T) {
	tests := []struct {
		name     string
		actual   []int
		expected []int
		want     bool
	}{
		{name: "single correct", actual: []int{1}, expected: []int{1}, want: true},
		{name: "single wrong", actual: []int{0}, expected: []int{1}, want: false},
		{name: "multi correct unordered", actual: []int{2, 1}, expected: []int{1, 2}, want: true},
		{name: "length mismatch", actual: []int{1, 2}, expected: []int{1}, want: false},
		{name: "empty", actual: nil, expected: nil, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsCorrectAnswer(tc.actual, tc.expected)
			if got != tc.want {
				t.Errorf("IsCorrectAnswer(%v, %v) = %v, want %v", tc.actual, tc.expected, got, tc.want)
			}
		})
	}
}
