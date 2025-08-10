package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

func (h *Handlers) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1,048,576
	// bytes (1MB).
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)

	// Initialize the json.Decoder, and call the DisallowUnknownFields() method on it
	// before decoding. If the JSON from the client includes any field that cannot be
	// mapped to the target destination, the decoder will return an error instead of
	// ignoring the field.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the request body to the destination.
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf(
				"body contains badly-formed JSON (at character %d)",
				syntaxError.Offset,
			)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf(
					"body contains incorrect JSON type for field %q",
					unmarshalTypeError.Field,
				)
			}
			return fmt.Errorf(
				"body contains incorrect JSON type (at character %d)",
				unmarshalTypeError.Offset,
			)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// If the JSON contains a field which cannot be mapped to the target destination
		// then Decode() will now return an error message in the format "json: unknown
		// field "<name>"". We check for this, extract the field name from the error,
		// size limit of 1MB and we return a clear error message.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

		// This indicates an error in our code.
		case errors.As(err, &invalidUnmarshalError):
			return err

		default:
			return err
		}
	}

	// Call Decode() again, using a pointer to an empty anonymous struct as the
	// destination. If the request body only contained a single JSON value this will
	// return an io.EOF error. So if we get anything else, we know that there is
	// additional data in the request body and we return our own custom error message.
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (h *Handlers) writeJSON(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	data envelope,
	headers http.Header,
) {
	js, err := json.MarshalIndent(data, "", "\t")
	if err == nil {
		js = append(js, '\n')
		maps.Copy(w.Header(), headers)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, err = w.Write(js)
	}

	if err != nil {
		// This indicates error in our code and should never happen. If it does, we log
		// it and send clients a plain 500 Internal Server Error.
		h.logError(r, err)
		w.WriteHeader(500)
	}
}

func (h *Handlers) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	idString := params.ByName("id")
	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidIDParam, idString)
	}

	return id, nil
}

// The readCSV() helper reads a string value from the query string and then splits it
// into a slice on the comma character. If no matching key could be found, it returns
// the provided default value.
func (h *Handlers) readCSV(qs url.Values, key string, defaultValue []string) []string {
	// Extract the value from the query string.
	csv := qs.Get(key)

	// If no key exists (or the value is empty) then return the default value.
	if csv == "" {
		return defaultValue
	}

	// Otherwise parse the value into a []string slice and return it.
	return strings.Split(csv, ",")
}

// The readInt() helper reads a string value from the query string and converts it to an
// integer before returning. If no matching key could be found it returns the provided
// default value. If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func (h *Handlers) readInt(
	qs url.Values,
	key string,
	defaultValue int,
	valErrs map[string]string,
) int {
	// Extract the value from the query string.
	s := qs.Get(key)

	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}

	// Try to convert the value to an int. If this fails, add an error message to the
	// validator instance and return the default value.
	i, err := strconv.Atoi(s)
	if err != nil {
		valErrs[key] = "must be an integer value"
		return defaultValue
	}

	// Otherwise, return the converted integer value.
	return i
}

// parseSortParams() method reads sort parameters from the query string and validates
// the values against a list. Any validation error is added to valErrs map
func (h *Handlers) validateSortFields(
	sortFields []string,
	safeList map[string]struct{},
	valErrs map[string]string,
) {
	if len(sortFields) > len(safeList) {
		valErrs["sort"] = "contains too many fields"
		return
	}

	for _, key := range sortFields {
		if _, ok := safeList[key]; !ok {
			valErrs[key] = "is not a valid sort field"
		}
	}
}
