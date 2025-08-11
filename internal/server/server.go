package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-redis/redis"
)

type Server struct {
	httpAddr  string
	redisAddr string
	Rdb       *redis.Client
	mux       *http.ServeMux
}

func NewServer(httpAddr, redisAddr string) (*Server, error) {
	s := &Server{
		httpAddr:  httpAddr,
		redisAddr: redisAddr,
		mux:       http.NewServeMux(),
	}

	// Static file server for html/css/js
	// fs := http.FileServer(http.Dir("./static"))
	// s.mux.Handle("/", fs)

	// API endpoints for paste CRUD
	s.mux.HandleFunc("GET /api/paste/{id}", s.getPasteAPIHandler)
	s.mux.HandleFunc("POST /api/paste", s.createPasteAPIHandler)
	s.mux.HandleFunc("DELETE /api/paste/{id}", s.deletePasteAPIHandler)

	// Endpoints for rendering html
	s.mux.HandleFunc("GET /", s.viewHandler)

	rdb := redis.NewClient(&redis.Options{
		Addr: s.redisAddr,
	})
	_, err := rdb.Ping().Result()
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		return nil, fmt.Errorf("failed to connect to redis database: %v", err)
	}
	s.Rdb = rdb

	return s, nil
}

func (s *Server) Serve() error {
	err := http.ListenAndServe(s.httpAddr, s.mux)
	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	return nil
}
