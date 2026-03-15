package match

import "testing"

func TestCanTransition(t *testing.T) {
	valid := [][2]MatchState{
		{StateCreated, StateLobby},
		{StateLobby, StateActive},
		{StateLobby, StateAbandoned},
		{StateActive, StateRevealing},
		{StateActive, StateLeaderboard},
		{StateActive, StateAbandoned},
		{StateRevealing, StateActive},
		{StateRevealing, StateScoring},
		{StateLeaderboard, StateActive},
		{StateLeaderboard, StateScoring},
		{StateScoring, StateEnded},
	}
	for _, pair := range valid {
		if !CanTransition(pair[0], pair[1]) {
			t.Errorf("expected CanTransition(%s, %s) = true", pair[0], pair[1])
		}
	}

	invalid := [][2]MatchState{
		{StateCreated, StateActive},
		{StateActive, StateEnded},
		{StateEnded, StateLobby},
		{StateAbandoned, StateActive},
	}
	for _, pair := range invalid {
		if CanTransition(pair[0], pair[1]) {
			t.Errorf("expected CanTransition(%s, %s) = false", pair[0], pair[1])
		}
	}
}

func TestMustTransition(t *testing.T) {
	if err := MustTransition(StateCreated, StateCreated); err != nil {
		t.Errorf("same state should be allowed: %v", err)
	}
	if err := MustTransition(StateCreated, StateLobby); err != nil {
		t.Errorf("valid transition should not error: %v", err)
	}
	if err := MustTransition(StateEnded, StateLobby); err == nil {
		t.Error("invalid transition should return error")
	}
}
