package server

import (
	"net/http"

	"github.com/lcrownover/paste/internal/storage"
)

func (s *Server) viewHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	// Home page
	if id == "" {
		if err := s.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Found paste ID in url
	paste, found, err := storage.GetPaste(s.Rdb, id)
	if err != nil {
		if err := s.templates.ExecuteTemplate(w, "error.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}
	if !found {
		if err := s.templates.ExecuteTemplate(w, "error.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}

	if err := s.templates.ExecuteTemplate(w, "paste.html", paste); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
