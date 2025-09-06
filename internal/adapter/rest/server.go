package rest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	srv    *http.Server
	logger *slog.Logger
}

func NewServer(srv *http.Server, logger *slog.Logger) *Server {
	return &Server{
		srv:    srv,
		logger: logger,
	}
}

func (s *Server) Run() error {
	s.logger.Info("server started",
		slog.String("port", s.srv.Addr),
	)

	err := s.srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop() error {
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.srv.Shutdown(timeout)
	if err != nil {
		return err
	}

	s.logger.Info("server stopped",
		slog.String("port", s.srv.Addr),
	)

	return err
}
