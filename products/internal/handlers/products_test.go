package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateProductHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	t.Run("create product successfully", func(t *testing.T) {
		productInput := productDTO{
			Name:        "Bluetooth Headphones",
			Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
			Price:       129.99,
			Quantity:    25,
		}

		body, _ := json.Marshal(productInput)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
		rw := httptest.NewRecorder()

		h := NewHandlers(logger)
		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var actualProduct productDTO
		err := json.NewDecoder(res.Body).Decode(&actualProduct)
		assert.NoError(t, err)
		assert.Equal(t, productInput, actualProduct)
	})

	t.Run(
		"respond with bad request if request body contains unknown field",
		func(t *testing.T) {
			var productInput = struct {
				ID          int     `json:"id"`
				Name        string  `json:"name"`
				Description string  `json:"description"`
				Price       float64 `json:"price"`
				Quantity    int     `json:"quantity"`
			}{
				ID:          1,
				Name:        "Bluetooth Headphones",
				Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
				Price:       129.99,
				Quantity:    25,
			}

			body, _ := json.Marshal(productInput)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
			rw := httptest.NewRecorder()

			h := NewHandlers(logger)
			h.CreateProductHandler(rw, req)
			res := rw.Result()
			defer res.Body.Close()

			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
			assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

			expectError := envelope{"error": `json: unknown field "id"`}

			var actualError envelope
			err := json.NewDecoder(res.Body).Decode(&actualError)
			assert.NoError(t, err)

			assert.Equal(t, expectError, actualError)
		},
	)

	t.Run("respond with bad request if request body is empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(""))
		rw := httptest.NewRecorder()

		h := NewHandlers(logger)
		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		expectError := envelope{"error": `body must not be empty`}

		var actualError envelope
		err := json.NewDecoder(res.Body).Decode(&actualError)
		assert.NoError(t, err)

		assert.Equal(t, expectError, actualError)
	})
}
