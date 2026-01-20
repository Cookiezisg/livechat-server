package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"livechat-server/internal/auth"
	"livechat-server/internal/http/middleware"
	"livechat-server/internal/store"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Store     *store.Store
	JWTSecret string
	TokenTTL  time.Duration
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Avatar   string `json:"avatarUrl"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if len(req.Password) < 6 || len(req.Username) < 3 || len(req.Email) < 5 {
		respondError(w, http.StatusBadRequest, "invalid credentials")
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	user, err := h.Store.CreateUser(r.Context(), strings.TrimSpace(req.Email), strings.TrimSpace(req.Username), string(passwordHash), strings.TrimSpace(req.Avatar))
	if err != nil {
		respondError(w, http.StatusConflict, "user already exists")
		return
	}
	token, err := auth.NewToken(user.ID, user.Username, h.JWTSecret, timeSecondsToDuration(h.TokenTTL))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not sign token")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	user, passwordHash, err := h.Store.GetUserByEmail(r.Context(), strings.TrimSpace(req.Email))
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := auth.NewToken(user.ID, user.Username, h.JWTSecret, timeSecondsToDuration(h.TokenTTL))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not sign token")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func (h AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	respondJSON(w, http.StatusOK, user)
}

func timeSecondsToDuration(duration time.Duration) time.Duration {
	if duration <= 0 {
		return 24 * time.Hour
	}
	return duration
}
