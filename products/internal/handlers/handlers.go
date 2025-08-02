package handlers

import "log/slog"

type Handlers struct {
	logger *slog.Logger
}

func NewHandlers(logger *slog.Logger) *Handlers {
	return &Handlers{logger: logger}
}
