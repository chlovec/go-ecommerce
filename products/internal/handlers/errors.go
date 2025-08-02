package handlers

import (
	"encoding/json"
	"net/http"
)

var invalidUnmarshalError *json.InvalidUnmarshalError

// The logError() method is a helper for logging an error message, along
// with the current request method and URL as attributes in the log entry.
func (h *Handlers) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	h.logger.Error(err.Error(), "method", method, "uri", uri)
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
	err = h.writeJSON(w, status, envelope{"error": message}, nil)
	if err != nil {
		h.logError(r, err)
		w.WriteHeader(500)
	}
}

func (h *Handlers) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.errorResponse(w, r, http.StatusBadRequest, err.Error(), err)
}
