package handlers

import (
	"encoding/json"
	"net/http"

	"livechat-server/internal/http/middleware"
	"livechat-server/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RoomsHandler struct {
	Store *store.Store
}

type createRoomRequest struct {
	Name      string `json:"name"`
	IsPrivate bool   `json:"isPrivate"`
}

func (h RoomsHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rooms, err := h.Store.ListRooms(r.Context(), user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load rooms")
		return
	}
	respondJSON(w, http.StatusOK, rooms)
}

func (h RoomsHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if len(req.Name) < 2 {
		respondError(w, http.StatusBadRequest, "room name too short")
		return
	}
	room, err := h.Store.CreateRoom(r.Context(), req.Name, req.IsPrivate, user.ID)
	if err != nil {
		respondError(w, http.StatusConflict, "room already exists")
		return
	}
	_ = h.Store.AddRoomMember(r.Context(), room.ID, user.ID, "owner")
	respondJSON(w, http.StatusCreated, room)
}

func (h RoomsHandler) Join(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Store.AddRoomMember(r.Context(), roomID, user.ID, "member"); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to join room")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (h RoomsHandler) Leave(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Store.RemoveRoomMember(r.Context(), roomID, user.ID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to leave room")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "left"})
}
