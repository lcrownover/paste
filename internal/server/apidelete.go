package server

import (
	"log/slog"
	"net/http"

	"github.com/lcrownover/paste/internal/storage"
)

func (s *Server) deletePasteAPIHandler(w http.ResponseWriter, r *http.Request) {
	pasteID := r.PathValue("id")
	if pasteID == "" {
		slog.Error("Paste ID not provided in URL params")
		http.Error(w, "Missing paste ID url param", http.StatusBadRequest)
		return
	}

	_, found, err := storage.GetPaste(s.Rdb, pasteID)
	if err != nil {
		slog.Error("Failed to get paste", "error", err)
		http.Error(w, "Failed to delete paste", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	slog.Info("Deleting paste", "id", pasteID)

	err = storage.DeletePaste(s.Rdb, pasteID)
	if err != nil {
		http.Error(w, "Failed to delete paste", http.StatusInternalServerError)
		return
	}

	slog.Info("Paste deleted successfully", "id", pasteID)
}
