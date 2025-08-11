package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/lcrownover/paste/internal/storage"
)

func (s *Server) createPasteAPIHandler(w http.ResponseWriter, r *http.Request) {
	var pasteRequest CreatePasteRequest
	err := json.NewDecoder(r.Body).Decode(&pasteRequest)
	if err != nil {
		slog.Error("failed to decode create body", "error", err)
		http.Error(w, "Invalid paste body", http.StatusBadRequest)
		return
	}

	paste, err := storage.CreatePaste(s.Rdb, pasteRequest.Content, pasteRequest.LifetimeSeconds)
	if err != nil {
		slog.Error("failed to create paste", "error", err)
		http.Error(w, "Failed to create paste", http.StatusInternalServerError)
		return
	}

	pasteResp := PasteResponse{
		ID:              paste.ID,
		Content:         paste.Content,
		LifetimeSeconds: paste.LifetimeSeconds,
	}

	resp, err := json.Marshal(pasteResp)
	if err != nil {
		slog.Error("failed to marshal paste response", "error", err)
		http.Error(w, "Failed to marshal paste response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}
