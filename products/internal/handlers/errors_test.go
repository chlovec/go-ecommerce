package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	testError := "request error"
	req := httptest.NewRequest(http.MethodGet, "/test/endpoint", nil)
	h := NewHandlers(logger)
	h.logError(req, errors.New(testError))

	assert.Contains(t, buf.String(), testError)
}

func TestErrorResponse(t *testing.T) {
	t.Run("should write error response successfully", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		h := NewHandlers(logger)
		req := httptest.NewRequest(http.MethodGet, "/test/endpoint", nil)
		rw := httptest.NewRecorder()

		expectedMsg := "test error message"
		testError := "test error"
		h.errorResponse(rw, req, http.StatusBadRequest, expectedMsg, errors.New(testError))

		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		var actualError envelope
		err := json.NewDecoder(res.Body).Decode(&actualError)
		assert.NoError(t, err)

		expectedErr := envelope{"error": expectedMsg}
		assert.Equal(t, expectedErr, actualError)
		assert.Contains(t, buf.String(), testError)
	})

	t.Run("should respond with 500 bad request if writeJson fails", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		h := NewHandlers(logger)
		req := httptest.NewRequest(http.MethodGet, "/test/endpoint", nil)
		rw := httptest.NewRecorder()

		type Node struct {
			Value string
			Next  *Node
		}
		v := &Node{Value: "A"}
		v.Next = v

		expectedMsg := envelope{"error": v}
		testError := "test error"
		h.errorResponse(rw, req, http.StatusBadRequest, expectedMsg, errors.New(testError))

		res := rw.Result()
		defer res.Body.Close()

		errMsg := `"json: unsupported value: encountered a cycle via *handlers.Node" method=GET uri=/test/endpoint`
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "", res.Header.Get("Content-Type"))
		assert.Contains(t, buf.String(), testError)
		assert.Contains(t, buf.String(), errMsg)
	})
}
