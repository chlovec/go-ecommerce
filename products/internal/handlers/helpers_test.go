package handlers

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func setup(t *testing.T, body string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()
	return w, req
}

func TestReadJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	h := Handlers{logger: logger}

	t.Run("valid JSON", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice","age":30}`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		w, req := setup(t, "")
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil || err.Error() != "body must not be empty" {
			t.Errorf("expected 'body must not be empty', got %v", err)
		}
	})

	t.Run("malformed JSON (syntax error)", func(t *testing.T) {
		w, req := setup(t, `{"name":`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil || !strings.Contains(err.Error(), "body contains badly-formed JSON") {
			t.Errorf("expected syntax error, got %v", err)
		}
	})

	t.Run("unexpected EOF", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice",`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil || !strings.Contains(err.Error(), "body contains badly-formed JSON") {
			t.Errorf("expected unexpected EOF error, got %v", err)
		}
	})

	t.Run("type mismatch", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice","age":"not-a-number"}`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil ||
			!strings.Contains(err.Error(), "body contains incorrect JSON type for field") {
			t.Errorf("expected unmarshal type error, got %v", err)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice","age":30,"extra":"field"}`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil || !strings.Contains(err.Error(), "json: unknown field") {
			t.Errorf("expected unknown field error, got %v", err)
		}
	})

	t.Run("too large body", func(t *testing.T) {
		largeName := strings.Repeat("a", 1_048_577) // 1MB + 1 byte
		largeBody := `{"name":"` + largeName + `","age":25}`

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeBody))
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil || !strings.Contains(err.Error(), "body must not be larger than") {
			t.Errorf("expected max bytes error, got %v", err)
		}
	})

	t.Run("multiple JSON values", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice","age":30}{"name":"Bob","age":40}`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil ||
			!strings.Contains(err.Error(), "body must only contain a single JSON value") {
			t.Errorf("expected multiple JSON values error, got %v", err)
		}
	})

	t.Run("badly formed json at character d", func(t *testing.T) {
		largeBody := strings.Repeat("a", 1_048_577) // 1MB + 1 byte
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(strings.NewReader(largeBody)))
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		if err == nil ||
			!strings.Contains(err.Error(), "body contains badly-formed JSON (at character 1)") {
			t.Errorf("body contains badly-formed JSON (at character 1) error, got %v", err)
		}
	})

	t.Run("incorrect json type at character d", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`"not-an-int"`))
		var dst int
		err := h.readJSON(w, req, &dst)
		if err == nil ||
			!strings.Contains(err.Error(), "body contains incorrect JSON type (at character 12)") {
			t.Errorf("body contains incorrect JSON type (at character), got %v", err)
		}
	})

	t.Run("invalid unmarshal non-pointer", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice"}`))
		var ts TestStruct
		err := h.readJSON(w, req, ts) // ts is not a pointer
		if err == nil || !strings.Contains(err.Error(), "json: Unmarshal(non-pointer") {
			t.Errorf("expected invalid unmarshal error, got %v", err)
		}
	})
}
