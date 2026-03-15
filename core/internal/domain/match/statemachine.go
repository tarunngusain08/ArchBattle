package match

import "fmt"

var validTransitions = map[MatchState]map[MatchState]struct{}{
	StateCreated: {
		StateLobby: {},
	},
	StateLobby: {
		StateActive:    {},
		StateAbandoned: {},
	},
	StateActive: {
		StateRevealing: {},
		StateAbandoned: {},
	},
	StateRevealing: {
		StateActive:  {},
		StateScoring: {},
	},
	StateScoring: {
		StateEnded: {},
	},
}

func CanTransition(from, to MatchState) bool {
	next, ok := validTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

func MustTransition(from, to MatchState) error {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid match transition %s -> %s", from, to)
	}
	return nil
}
