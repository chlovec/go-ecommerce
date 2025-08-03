package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type mockFieldError struct {
	field string
	tag   string
	param string
	err   string
}

func (m mockFieldError) Field() string                        { return m.field }
func (m mockFieldError) Tag() string                          { return m.tag }
func (m mockFieldError) Param() string                        { return m.param }
func (m mockFieldError) Error() string                        { return m.err }
func (m mockFieldError) ActualTag() string                    { return m.tag } // Added method
func (m mockFieldError) Namespace() string                    { return m.field }
func (m mockFieldError) StructNamespace() string              { return m.field }
func (m mockFieldError) StructField() string                  { return m.field }
func (m mockFieldError) Kind() reflect.Kind                   { return reflect.String }
func (m mockFieldError) Type() reflect.Type                   { return reflect.TypeOf("") }
func (m mockFieldError) Value() interface{}                   { return "" }
func (m mockFieldError) Translate(trans ut.Translator) string { return m.err }

type FaultyResponseWriter struct {
	http.ResponseWriter
}

func (w *FaultyResponseWriter) Write(b []byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	testError := "request error"
	req := httptest.NewRequest(http.MethodGet, "/test/endpoint", nil)
	h := NewHandlers(logger, &sql.DB{})
	h.logError(req, errors.New(testError))

	assert.Contains(t, buf.String(), testError)
}

func TestErrorResponse(t *testing.T) {
	db := sql.DB{}
	t.Run("should write error response successfully", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		h := NewHandlers(logger, &db)
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

		h := NewHandlers(logger, &db)
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

func TestGetJSONName(t *testing.T) {
	jsonFieldMaps := map[string]string{
		"Name":       "name",
		"CategoryID": "category_id",
	}

	testData := []struct {
		name     string
		key      string
		expected string
	}{
		{"key present", "Name", "name"},
		{"key not present", "Cat", "Cat"},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			actual := getJsonName(tt.key, jsonFieldMaps)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetValidationMessage(t *testing.T) {
	tests := []struct {
		name     string
		fieldErr validator.FieldError
		expected string
	}{
		{
			"Required field",
			mockFieldError{"Username", "required", "", ""},
			"is required",
		},
		{
			"Invalid email",
			mockFieldError{"Email", "email", "", ""},
			"is not a valid email address",
		},
		{
			"Min length",
			mockFieldError{"Password", "min", "8", ""},
			"must be at least 8 characters long",
		},
		{
			"Max length",
			mockFieldError{"Username", "max", "20", ""},
			"must be at most 20 characters long",
		},
		{
			"Exact length",
			mockFieldError{"Code", "len", "6", ""},
			"must be exactly 6 characters long",
		},
		{
			"Equal",
			mockFieldError{"Age", "eq", "18", ""},
			"must be equal to 18",
		},
		{
			"Not Equal",
			mockFieldError{"Role", "ne", "admin", ""},
			"must not be equal to admin",
		},
		{
			"Less Than",
			mockFieldError{"Amount", "lt", "100", ""},
			"must be less than 100",
		},
		{
			"Less Than or Equal",
			mockFieldError{"Amount", "lte", "50", ""},
			"must be less than or equal to 50",
		},
		{
			"Greater Than",
			mockFieldError{"Score", "gt", "10", ""},
			"must be greater than 10",
		},
		{
			"Greater Than or Equal",
			mockFieldError{"Score", "gte", "20", ""},
			"must be greater than or equal to 20",
		},
		{
			"OneOf",
			mockFieldError{"Status", "oneof", "pending approved rejected", ""},
			"must be one of [pending approved rejected]",
		},
		{"Valid URL", mockFieldError{"Website", "url", "", ""}, "must be a valid URL"},
		{"Valid UUID", mockFieldError{"UserID", "uuid", "", ""}, "must be a valid UUID"},
		{
			"Alphanumeric",
			mockFieldError{"Username", "alphanum", "", ""},
			"must contain only alphanumeric characters",
		},
		{"Numeric", mockFieldError{"Price", "numeric", "", ""}, "must be a valid number"},
		{
			"Boolean",
			mockFieldError{"IsActive", "boolean", "", ""},
			"must be a boolean value",
		},
		{
			"Datetime",
			mockFieldError{"CreatedAt", "datetime", "2006-01-02", ""},
			"must be a valid datetime format (2006-01-02)",
		},
		{
			"Unknown Tag",
			mockFieldError{"Custom", "custom_rule", "", "Custom validation error"},
			"failed validation: Custom validation error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getFieldErrorMessage(tc.fieldErr)
			assert.Equal(t, tc.expected, result)
		})
	}
}
