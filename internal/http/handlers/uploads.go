package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"livechat-server/internal/http/middleware"

	"github.com/google/uuid"
)

type UploadsHandler struct {
	UploadDir   string
	MaxUploadMB int64
}

func (h UploadsHandler) Upload(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := os.MkdirAll(h.UploadDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create upload dir")
		return
	}
	maxBytes := h.MaxUploadMB * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file missing")
		return
	}
	defer file.Close()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}
	id := uuid.New().String()
	filename := id + strings.ToLower(ext)
	path := filepath.Join(h.UploadDir, filename)
	out, err := os.Create(path)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to write file")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]string{"url": "/uploads/" + filename})
}
