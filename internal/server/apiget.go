package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/lcrownover/paste/internal/storage"
)

func (s *Server) getPasteAPIHandler(w http.ResponseWriter, r *http.Request) {
	pasteID := r.PathValue("id")
	if pasteID == "" {
		slog.Error("failed to get paste id from url params")
		http.Error(w, "Paste ID not provided in url params", http.StatusBadRequest)
		return
	}
	paste, found, err := storage.GetPaste(s.Rdb, pasteID)
	if !found {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("server failed to get paste", "error", err)
		http.Error(w, "Failed to get paste", http.StatusInternalServerError)
		return
	}

	slog.Info("Paste retrieved successfully", "id", pasteID)

	ret, err := json.Marshal(paste)
	if err != nil {
		slog.Error("failed to marshal paste response", "error", err)
		http.Error(w, "Failed to marshal paste response", http.StatusInternalServerError)
		return
	}

	w.Write(ret)
}
