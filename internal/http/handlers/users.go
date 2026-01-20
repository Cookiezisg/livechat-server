package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"livechat-server/internal/store"
)

type UsersHandler struct {
	Store *store.Store
}

func (h UsersHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	users, err := h.Store.SearchUsers(r.Context(), query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load users")
		return
	}
	respondJSON(w, http.StatusOK, users)
}
