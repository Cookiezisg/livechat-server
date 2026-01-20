package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func parseTimeParam(raw string) time.Time {
	if raw == "" {
		return time.Now().Add(1 * time.Second)
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Now().Add(1 * time.Second)
	}
	return parsed
}
