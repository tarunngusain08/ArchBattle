package match

import "testing"

func TestCalculateDelta(t *testing.T) {
	tests := []struct {
		name      string
		input     RatingInput
		wantRange [2]int // [min, max] inclusive
	}{
		{
			name: "new player beats higher-rated opponent",
			input: RatingInput{
				CurrentRating:      1000,
				MatchesPlayed:      0,
				PerformanceScore:   500,
				MeanOpponentPerf:   300,
				MeanOpponentRating: 1200,
			},
			// New player (k=32), outperforms opponent, positive delta
			wantRange: [2]int{5, 32},
		},
		{
			name: "experienced player loses to equal",
			input: RatingInput{
				CurrentRating:      1500,
				MatchesPlayed:      50,
				PerformanceScore:   100,
				MeanOpponentPerf:   400,
				MeanOpponentRating: 1500,
			},
			// Experienced (k=16), underperforms, negative delta
			wantRange: [2]int{-16, -1},
		},
		{
			name: "disconnect tax halves the delta",
			input: RatingInput{
				CurrentRating:      1000,
				MatchesPlayed:      5,
				PerformanceScore:   0,
				MeanOpponentPerf:   300,
				MeanOpponentRating: 1000,
				ApplyDisconnectTax: true,
			},
			// Should produce a smaller-magnitude delta than without tax
			wantRange: [2]int{-16, 0},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateDelta(tc.input)
			if got < tc.wantRange[0] || got > tc.wantRange[1] {
				t.Errorf("CalculateDelta() = %d, want in [%d, %d]", got, tc.wantRange[0], tc.wantRange[1])
			}
		})
	}
}
