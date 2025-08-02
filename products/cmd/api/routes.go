package main

import (
	"net/http"

	"github.com/chlovec/go-ecommerce/products/internal/handlers"
	"github.com/julienschmidt/httprouter"
)

func routes() http.Handler {
	router := httprouter.New()

	h := handlers.NewHandlers()

	// Products request routing
	router.HandlerFunc(http.MethodPost, "/v1/api/products", h.CreateProductHandler)

	return  router
}