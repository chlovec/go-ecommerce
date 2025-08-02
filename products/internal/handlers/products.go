package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// POST v1/api/products
func (h *Handlers) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var product = struct {
		ID          int
		Name        string
		Description string
		Price       float64
		Quantity    int
		Version     int
		CreatedAT   time.Time
	}{
		ID:          1,
		Name:        "Bluetooth Headphones",
		Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
		Price:       129.99,
		Quantity:    25,
		Version:     1,
		CreatedAT:   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}