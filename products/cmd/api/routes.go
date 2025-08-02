package main

import (
	"log/slog"
	"net/http"

	"github.com/chlovec/go-ecommerce/products/internal/handlers"
	"github.com/julienschmidt/httprouter"
)

func routes(logger *slog.Logger) http.Handler {
	router := httprouter.New()

	h := handlers.NewHandlers(logger)

	// Products request routing
	router.HandlerFunc(http.MethodPost, "/v1/api/products", h.CreateProductHandler)

	return router
}
