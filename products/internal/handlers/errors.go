package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var fieldJSONMap = map[string]string{
	"Name":        "name",
	"CategoryID":  "category_id",
	"Description": "description",
	"Price":       "price",
	"Quantity":    "quantity",
}

func (h *Handlers) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.errorResponse(w, r, http.StatusBadRequest, err.Error(), err)
}

func (h *Handlers) failedValidationResponse(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	validationMessages := getValidationMessages(err)
	h.errorResponse(w, r, http.StatusUnprocessableEntity, validationMessages, err)
}

// The notFoundResponse() method will be used to send a 404 Not Found status code and
// JSON response to the client.
// func (h *Handlers) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
// 	message := "the requested resource could not be found"
// 	h.errorResponse(w, r, http.StatusNotFound, message, err)
// }

// The serverErrorResponse() method will be used when our handlers encounter an
// unexpected problem at runtime. It logs the detailed error message, then uses the
// errorResponse() helper to send a 500 Internal Server Error status code and JSON
// response (containing a generic error message) to the client.
func (h *Handlers) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	message := "the server encountered a problem and could not process your request"
	h.errorResponse(w, r, http.StatusInternalServerError, message, err)
}

// The errorResponse() method is a helper for sending JSON-formatted error
// messages to the client with a given status code.
func (h *Handlers) errorResponse(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	message any,
	err error,
) {
	// Log the error
	h.logError(r, err)

	// Write the response using the writeJSON() helper. If it return an error then log
	// it, and fall back to sending the client an empty response with a 500 Internal
	// Server Error status code.
	h.writeJSON(w, r, status, envelope{"error": message}, nil)
}

// The logError() method is a helper for logging an error message, along
// with the current request method and URL as attributes in the log entry.
func (h *Handlers) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	h.logger.Error(err.Error(), "method", method, "uri", uri)
}

func getValidationMessages(err error) map[string]string {
	validationErrors := make(map[string]string)
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, err := range ve {
			key := getJsonName(err.Field(), fieldJSONMap)
			validationErrors[key] = getFieldErrorMessage(err)
		}
	}
	return validationErrors
}

func getJsonName(key string, fieldMap map[string]string) string {
	if val, ok := fieldMap[key]; ok {
		return val
	}
	return key
}

func getFieldErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "is not a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters long", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters long", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", fe.Param())
	case "eq":
		return fmt.Sprintf("must be equal to %s", fe.Param())
	case "ne":
		return fmt.Sprintf("must not be equal to %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", fe.Param())
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "numeric":
		return "must be a valid number"
	case "boolean":
		return "must be a boolean value"
	case "datetime":
		return fmt.Sprintf("must be a valid datetime format (%s)", fe.Param())
	default:
		return fmt.Sprintf("failed validation: %s", fe.Error())
	}
}
