package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type APIServer interface {
	Serve() error
}

type HTTPServer interface {
	Shutdown(ctx context.Context) error
	ListenAndServe() error
}

type Server struct {
	addr       string
	config     config
	httpServer HTTPServer
	logger     *slog.Logger
}

func newServer(cfg config, logger *slog.Logger) APIServer {
	addr := fmt.Sprintf(":%d", cfg.port)
	return &Server{
		addr:   addr,
		config: cfg,
		logger: logger,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      routes(),
			IdleTimeout:  cfg.idleTimeout,
			ReadTimeout:  cfg.readTimeout,
			WriteTimeout: cfg.WriteTimeout,
			ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
	}
}

func (svr *Server) Serve() error {
	// Create a shutdownError channel. We will use this to receive any errors returned
	// by the graceful Shutdown() function.
	shutdownError := make(chan error)

	// Start a background goroutine.
	go func() {
		// Intercept the signals
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		svr.logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Call Shutdown() on the server like before, but now we only send on the
		// shutdownError channel if it returns an error.
		err := svr.httpServer.Shutdown(ctx)
		shutdownError <- err
	}()

	svr.logger.Info("starting server", "addr", svr.addr, "env", svr.config.env)

	err := svr.httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	svr.logger.Info("stopped server", "addr", svr.addr)

	return nil
}
