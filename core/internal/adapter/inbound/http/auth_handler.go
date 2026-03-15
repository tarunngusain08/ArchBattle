package http

import (
	"errors"
	stdhttp "net/http"
	"strings"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
)

type AuthHandler struct {
	service *domainauth.Service
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthHandler(service *domainauth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req registerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	result, err := h.service.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, result)
}

func (h *AuthHandler) Login(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	result, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		status := stdhttp.StatusBadRequest
		if errors.Is(err, domainauth.ErrInvalidCredentials) {
			status = stdhttp.StatusUnauthorized
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, result)
}

func (h *AuthHandler) Logout(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "missing bearer token"})
		return
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}
