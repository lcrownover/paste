package server

import (
	"html/template"
	"net/http"

	"github.com/lcrownover/paste/internal/storage"
)

func (s *Server) viewHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	// Home page
	if id == "" {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Found paste ID in url
	paste, found, err := storage.GetPaste(s.Rdb, id)
	if err != nil {
		tmpl := template.Must(template.ParseFiles("templates/error.html"))
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}
	if !found {
		tmpl := template.Must(template.ParseFiles("templates/error.html"))
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/paste.html"))
	if err := tmpl.Execute(w, paste); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
