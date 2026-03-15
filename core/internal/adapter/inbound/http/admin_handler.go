package http

import (
	"context"
	stdhttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domaindaily "github.com/radhakrishna/archbattle/core/internal/domain/daily"
	domaindiscussion "github.com/radhakrishna/archbattle/core/internal/domain/discussion"
	domainquestion "github.com/radhakrishna/archbattle/core/internal/domain/question"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

// DraftQuestioner is a port used by the admin handler to request AI-drafted questions.
type DraftQuestioner interface {
	DraftQuestion(ctx context.Context, req domainquestion.DraftRequest) (map[string]any, error)
}

// DiscussionSummarizer generates AI summaries from discussion entries.
type DiscussionSummarizer interface {
	SummarizeDiscussion(ctx context.Context, date string, entries []DiscussionEntryInput) (string, error)
}

// DiscussionEntryInput is the input format for the summarizer.
type DiscussionEntryInput struct {
	QuestionNumber  int
	Username        string
	ReasoningText   string
	AlternativeText string
	SurpriseText    string
}

type AdminHandler struct {
	questions  *domainquestion.Service
	daily      *domaindaily.Service
	ai         DraftQuestioner
	discussion *domaindiscussion.Service
	summarizer DiscussionSummarizer
}

type updateQuestionStatusRequest struct {
	Status     string `json:"status"`
	Rationale  string `json:"rationale"`
}

type draftQuestionRequest struct {
	Topic string `json:"topic"`
	Tier  string `json:"tier"`
	Mode  string `json:"mode"`
	Seed  string `json:"seed"`
}

type createQuestionRequest struct {
	Prompt         string   `json:"prompt"`
	Options        []string `json:"options"`
	CorrectAnswers []int    `json:"correctAnswers"`
	Rationale      string   `json:"rationale"`
	Topic          string   `json:"topic"`
	Tier           string   `json:"tier"`
	Mode           string   `json:"mode"`
}

func NewAdminHandler(questions *domainquestion.Service, daily *domaindaily.Service, ai DraftQuestioner, discussion *domaindiscussion.Service, summarizer DiscussionSummarizer) *AdminHandler {
	return &AdminHandler{questions: questions, daily: daily, ai: ai, discussion: discussion, summarizer: summarizer}
}

func (h *AdminHandler) ListDrafts(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "draft"
	}
	questions, err := h.questions.ListByStatus(r.Context(), status)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"questions": questions})
}

func (h *AdminHandler) UpdateQuestionStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid question id"})
		return
	}
	var req updateQuestionStatusRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	if req.Rationale != "" {
		if err := h.questions.UpdateRationale(r.Context(), questionID, req.Rationale); err != nil {
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.Status != "" {
		if err := h.questions.UpdateStatus(r.Context(), questionID, req.Status); err != nil {
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}

func (h *AdminHandler) ListDisputes(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	questions, err := h.questions.ListByStatus(r.Context(), "quarantined")
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"questions": questions})
}

func (h *AdminHandler) DraftQuestion(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.ai == nil {
		writeJSON(w, stdhttp.StatusNotImplemented, map[string]string{"error": "ai draft client not configured"})
		return
	}
	var req draftQuestionRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	draft, err := h.ai.DraftQuestion(r.Context(), domainquestion.DraftRequest{Topic: req.Topic, Tier: req.Tier, Mode: req.Mode, Seed: req.Seed})
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, draft)
}

func (h *AdminHandler) CreateQuestion(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req createQuestionRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	if req.Prompt == "" || len(req.Options) == 0 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "prompt and options required"})
		return
	}
	topic := shared.NormalizeTopic(req.Topic)
	if topic == "" {
		topic = "caching"
	}
	tier, _ := shared.ParseTier(req.Tier)
	if tier == "" {
		tier = shared.TierJunior
	}
	mode := shared.Mode(req.Mode)
	if mode == "" {
		mode = shared.ModeFFF
	}
	q := &domainquestion.Question{
		ID:             uuid.New(),
		Mode:           mode,
		Topic:          topic,
		DifficultyTier: tier,
		Prompt:         req.Prompt,
		Options:        req.Options,
		CorrectAnswers: req.CorrectAnswers,
		Rationale:      req.Rationale,
		Status:         "draft",
		IsActive:       false,
		DailyEligible:  false,
		CreatedAt:      time.Now().UTC(),
	}
	if err := h.questions.Create(r.Context(), q); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, map[string]any{"id": q.ID, "status": q.Status})
}

func (h *AdminHandler) PublishDaily(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	date := time.Now().UTC()
	if raw := r.URL.Query().Get("date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid date format"})
			return
		}
		date = parsed
	}
	challenge, err := h.daily.Publish(r.Context(), date)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, challenge)
}

func (h *AdminHandler) GenerateDailySummary(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.discussion == nil || h.summarizer == nil {
		writeJSON(w, stdhttp.StatusNotImplemented, map[string]string{"error": "discussion summary not configured"})
		return
	}
	date := time.Now().UTC().AddDate(0, 0, -1)
	if raw := r.URL.Query().Get("date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid date format (use YYYY-MM-DD)"})
			return
		}
		date = parsed
	}
	entries, err := h.discussion.List(r.Context(), date)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	inputs := make([]DiscussionEntryInput, len(entries))
	for i, e := range entries {
		inputs[i] = DiscussionEntryInput{
			QuestionNumber:  e.QuestionNumber,
			Username:        e.Username,
			ReasoningText:   e.ReasoningText,
			AlternativeText: e.AlternativeText,
			SurpriseText:    e.SurpriseText,
		}
	}
	summary, err := h.summarizer.SummarizeDiscussion(r.Context(), date.Format("2006-01-02"), inputs)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if err := h.daily.UpdateAISummary(r.Context(), date, summary); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok", "summary": summary})
}
