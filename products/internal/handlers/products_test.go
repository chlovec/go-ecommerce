package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateProductHandler(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil))

	h := NewHandlers()

	t.Run("should create product successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(""))
		rw := httptest.NewRecorder()

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		type product struct {
			ID          int       `json:"id"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
			Price       float64   `json:"price"`
			Quantity    int       `json:"quantity"`
			Version     int       `json:"version"`
			CreatedAT   time.Time `json:"created_at"`
		}

		expectedProduct := product{
			ID:          1,
			Name:        "Bluetooth Headphones",
			Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
			Price:       129.99,
			Quantity:    25,
			Version:     1,
			CreatedAT:   time.Date(2025, time.May, 25, 0, 0, 0, 0, time.UTC),
		}

		var actualProduct product
		err := json.NewDecoder(res.Body).Decode(&actualProduct)
		assert.NoError(t, err)
		assert.Equal(t, expectedProduct, actualProduct)
	})
}
