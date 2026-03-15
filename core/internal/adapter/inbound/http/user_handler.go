package http

import (
	stdhttp "net/http"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
	outboundpostgres "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/postgres"
)

type UserHandler struct {
	service   *domainauth.Service
	matchRepo *outboundpostgres.MatchRepo
	dailyRepo *outboundpostgres.DailyRepo
}

func NewUserHandler(service *domainauth.Service, matchRepo *outboundpostgres.MatchRepo, dailyRepo *outboundpostgres.DailyRepo) *UserHandler {
	return &UserHandler{service: service, matchRepo: matchRepo, dailyRepo: dailyRepo}
}

func (h *UserHandler) Me(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing user context"})
		return
	}
	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp := map[string]any{
		"id": user.ID, "username": user.Username, "email": user.Email, "tier": user.Tier,
		"juniorElo": user.JuniorELO, "seniorElo": user.SeniorELO, "staffElo": user.StaffELO,
		"matchesPlayed": user.MatchesPlayed, "currentStreak": user.CurrentStreak,
		"longestStreak": user.LongestStreak, "lastDailyDate": user.LastDailyDate, "createdAt": user.CreatedAt,
	}
	if h.matchRepo != nil {
		if history, err := h.matchRepo.UserMatchHistory(r.Context(), userID, 10); err == nil {
			matchHistory := make([]map[string]any, len(history))
			for i, e := range history {
				matchHistory[i] = map[string]any{"opponent": e.Opponent, "score": e.Score, "eloDelta": e.ELODelta}
			}
			resp["matchHistory"] = matchHistory
		}
		if topicStats, err := h.matchRepo.UserTopicStats(r.Context(), userID); err == nil {
			stats := make([]map[string]any, len(topicStats))
			for i, s := range topicStats {
				pct := 0.0
				if s.Total > 0 {
					pct = float64(s.Correct) / float64(s.Total) * 100
				}
				stats[i] = map[string]any{"topic": s.Topic, "correct": s.Correct, "total": s.Total, "accuracy": pct}
			}
			resp["topicStats"] = stats
		}
	}
	if h.dailyRepo != nil {
		if dates, err := h.dailyRepo.RecentDailyDates(r.Context(), userID, 30); err == nil {
			dateStrs := make([]string, len(dates))
			for i, d := range dates {
				dateStrs[i] = d.Format("2006-01-02")
			}
			resp["streakCalendar"] = dateStrs
		}
	}
	writeJSON(w, stdhttp.StatusOK, resp)
}
