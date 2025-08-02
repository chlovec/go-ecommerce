package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// POST v1/api/products
func (h *Handlers) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var product = struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Price       float64   `json:"price"`
		Quantity    int       `json:"quantity"`
		Version     int       `json:"version"`
		CreatedAT   time.Time `json:"created_at"`
	}{
		ID:          1,
		Name:        "Bluetooth Headphones",
		Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
		Price:       129.99,
		Quantity:    25,
		Version:     1,
		CreatedAT:   time.Date(2025, time.May, 25, 0, 0, 0, 0, time.UTC),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}
