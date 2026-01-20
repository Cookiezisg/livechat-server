package handlers

import (
	"net/http"
	"strconv"

	"livechat-server/internal/http/middleware"
	"livechat-server/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MessagesHandler struct {
	Store *store.Store
}

func (h MessagesHandler) RoomHistory(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "roomId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid room id")
		return
	}
	member, err := h.Store.IsRoomMember(r.Context(), roomID, user.ID)
	if err != nil || !member {
		respondError(w, http.StatusForbidden, "not a member")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	before := parseTimeParam(r.URL.Query().Get("before"))
	messages, err := h.Store.ListRoomMessages(r.Context(), roomID, limit, before)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}
	respondJSON(w, http.StatusOK, messages)
}

func (h MessagesHandler) DMHistory(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	peerID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	before := parseTimeParam(r.URL.Query().Get("before"))
	messages, err := h.Store.ListDMs(r.Context(), user.ID, peerID, limit, before)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}
	respondJSON(w, http.StatusOK, messages)
}
