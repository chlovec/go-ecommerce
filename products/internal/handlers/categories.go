package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/chlovec/go-ecommerce/products/internal/data"
)

type categoryDTO struct {
	Name        string `json:"name"        validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"omitempty"`
}

// POST v1/api/categories
func (h *Handlers) CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body. If it fails, respond with 400 Bad Request. Include a user
	// friendly message with the response.
	var payload categoryDTO

	err := h.readJSON(w, r, &payload)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	// Validate the request body. If validation fails, respond with 422 Unprocessable
	// Entity. Include a user friendly message with the response.
	err = h.validator.Struct(payload)
	if err != nil {
		h.failedValidationResponse(w, r, err)
		return
	}

	// Save category to db. If category id does not exist, send 400 Bad Request to the
	// client. For any other error, respond send 500 Internal Server Error.
	category := data.Category{
		Name:        payload.Name,
		Description: payload.Description,
	}

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.models.Category.Insert(ctx, &category)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	// Write successful response if all succeeds.
	// Create a location header to be included in the http response to
	// let the client know which url they can find newly created
	// resource at.
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/api/categories/%d", category.ID))
	h.writeJSON(w, r, http.StatusCreated, envelope{"category": category}, headers)
}

// GET v1/api/categories/{id}
func (h *Handlers) GetCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Read and validate id param.
	id, err := h.readIDParam(r)
	if err != nil || id < 1 {
		if id < 1 && err == nil {
			err = fmt.Errorf("%w: %d", ErrInvalidIDParam, id)
		}
		h.badRequestResponse(w, r, err)
		return
	}

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	category, err := h.models.Category.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			h.notFoundResponse(w, r, err)
		} else {
			h.serverErrorResponse(w, r, err)
		}
		return
	}

	env := envelope{"category": category}
	h.writeJSON(w, r, http.StatusOK, env, nil)
}

// GET /v1/api/categories?name={name}&page={page}&page_size={page_size}&sort={sort}
func (h *Handlers) ListCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// parse query params
	var filters data.Filters
	qs := r.URL.Query()
	valErrs := map[string]string{}

	filters.DateFrom = h.readTime(qs, "date_from", nil, valErrs)
	filters.DateTo = h.readTime(qs, "date_to", nil, valErrs)
	filters.IDs = h.readInt64Slice(qs, "id", nil, valErrs)
	filters.Name = qs.Get("name")
	filters.Sorts = h.readCSV(qs, "sort", []string{})
	filters.Page = h.readInt(qs, "page", 1, valErrs)
	filters.PageSize = h.readInt(qs, "page_size", 20, valErrs)

	if len(valErrs) > 0 {
		h.errorResponse(w, r, http.StatusBadRequest, valErrs, createErr(valErrs))
		return
	}

	// Validate
	err := h.validator.Struct(filters)
	if err != nil {
		h.failedValidationResponse(w, r, err)
		return
	}

	// call CategoryModel.GetAll to fetch categories
	// pass a context with a 5-second timeout deadline to ensure the query does not run forever.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	categories, metadata, err := h.models.Category.GetAll(ctx, filters)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	// write response
	env := envelope{"categories": categories, "metadata": metadata}
	h.writeJSON(w, r, http.StatusOK, env, nil)
}
