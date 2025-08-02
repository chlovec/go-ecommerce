package handlers

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		var actualResult TestStruct
		err := h.readJSON(w, req, &actualResult)
		assert.NoError(t, err)

		expectedResult := TestStruct{Name: "Alice", Age: 30}
		assert.Equal(t, expectedResult, actualResult)
	})

	t.Run("empty body", func(t *testing.T) {
		w, req := setup(t, "")
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		assert.Error(t, err)
		assert.Equal(t, "body must not be empty", err.Error())
	})

	t.Run("malformed JSON (syntax error)", func(t *testing.T) {
		w, req := setup(t, `{"name":`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		assert.Error(t, err)
		assert.Equal(t, "body contains badly-formed JSON", err.Error())
	})

	t.Run("type mismatch", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice","age":"not-a-number"}`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		assert.Error(t, err)
		assert.Equal(t, `body contains incorrect JSON type for field "age"`, err.Error())
	})

	t.Run("unknown field", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice","age":30,"extra":"field"}`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		assert.Error(t, err)
		assert.Equal(t, `json: unknown field "extra"`, err.Error())
	})

	t.Run("too large body", func(t *testing.T) {
		largeName := strings.Repeat("a", 1_048_577) // 1MB + 1 byte
		largeBody := `{"name":"` + largeName + `","age":25}`

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeBody))
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		assert.Error(t, err)
		assert.Equal(t, "body must not be larger than 1048576 bytes", err.Error())
	})

	t.Run("multiple JSON values", func(t *testing.T) {
		w, req := setup(t, `{"name":"Alice","age":30}{"name":"Bob","age":40}`)
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		assert.Error(t, err)
		assert.Equal(t, "body must only contain a single JSON value", err.Error())
	})

	t.Run("badly formed json at character d", func(t *testing.T) {
		largeBody := strings.Repeat("a", 1_048_577) // 1MB + 1 byte
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(strings.NewReader(largeBody)))
		var ts TestStruct
		err := h.readJSON(w, req, &ts)
		assert.Error(t, err)
		assert.Equal(t, "body contains badly-formed JSON (at character 1)", err.Error())
	})

	t.Run("incorrect json type at character d", func(t *testing.T) {
		w, req := setup(t, `"not-an-int"`)
		var dst int
		err := h.readJSON(w, req, &dst)
		assert.Error(t, err)
		assert.Equal(t, "body contains incorrect JSON type (at character 12)", err.Error())
	})

	t.Run("invalid unmarshal non-pointer", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Alice"}`))
		var ts TestStruct
		err := h.readJSON(w, req, ts) // ts is not a pointer
		assert.Error(t, err)
		assert.Equal(t, "json: Unmarshal(non-pointer handlers.TestStruct)", err.Error())
	})
}
