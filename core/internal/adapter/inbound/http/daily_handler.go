package http

import (
	stdhttp "net/http"
	"strconv"
	"time"

	domaindaily "github.com/radhakrishna/archbattle/core/internal/domain/daily"
)

type DailyHandler struct {
	service *domaindaily.Service
}

type dailySubmitRequest struct {
	Date        string           `json:"date"`
	Answers     map[string][]int `json:"answers"`
	TotalMillis int64            `json:"totalMillis"`
}

func NewDailyHandler(service *domaindaily.Service) *DailyHandler {
	return &DailyHandler{service: service}
}

func (h *DailyHandler) GetChallenge(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	date := time.Now().UTC()
	if raw := r.URL.Query().Get("date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid date format"})
			return
		}
		date = parsed
	}
	challenge, err := h.service.GetChallenge(r.Context(), date)
	if err != nil {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, challenge)
}

func (h *DailyHandler) Submit(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing user context"})
		return
	}
	var req dailySubmitRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	date := time.Now().UTC()
	if req.Date != "" {
		parsed, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid date format"})
			return
		}
		date = parsed
	}
	result, err := h.service.Submit(r.Context(), domaindaily.Submission{UserID: userID, ChallengeDate: date, Answers: req.Answers, TotalMillis: req.TotalMillis})
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, result)
}

func (h *DailyHandler) ShareCard(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	score, err := strconv.Atoi(r.URL.Query().Get("score"))
	if err != nil || score < 0 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid score"})
		return
	}
	correct, err := strconv.Atoi(r.URL.Query().Get("correct"))
	if err != nil || correct < 0 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid correct"})
		return
	}
	total, err := strconv.Atoi(r.URL.Query().Get("total"))
	if err != nil || total < 0 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid total"})
		return
	}
	percentile, err := strconv.ParseFloat(r.URL.Query().Get("percentile"), 64)
	if err != nil || percentile < 0 || percentile > 100 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid percentile"})
		return
	}
	date := time.Now().UTC()
	if raw := r.URL.Query().Get("date"); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			date = parsed
		}
	}
	text := h.service.GenerateShareCard(domaindaily.Result{ChallengeDate: date, Score: score, Percentile: percentile}, correct, total)
	writeJSON(w, stdhttp.StatusOK, map[string]string{"shareCardText": text})
}
