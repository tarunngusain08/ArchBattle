package match

import "math"

type RatingInput struct {
	CurrentRating      int
	MatchesPlayed      int
	PerformanceScore   float64
	MeanOpponentPerf   float64
	MeanOpponentRating float64
	ApplyDisconnectTax bool
}

func CalculateDelta(input RatingInput) int {
	expectedScore := 1 / (1 + math.Pow(10, (input.MeanOpponentRating-float64(input.CurrentRating))/400))
	relativePerfNorm := (input.PerformanceScore - input.MeanOpponentPerf + 1) / 2
	if relativePerfNorm < 0 {
		relativePerfNorm = 0
	}
	if relativePerfNorm > 1 {
		relativePerfNorm = 1
	}

	k := 16.0
	if input.MatchesPlayed < 30 {
		k = 32.0
	}

	delta := int(math.Round(k * (relativePerfNorm - expectedScore)))
	if input.ApplyDisconnectTax {
		delta = int(math.Round(float64(delta) / 2))
	}
	return delta
}
