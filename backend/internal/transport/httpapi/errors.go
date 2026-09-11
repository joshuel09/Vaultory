// Package httpapi is Vaultory's HTTP transport. It decodes requests, calls the application
// services, and renders responses. It holds no business rules (Constitution Principle II).
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Error codes carried to the client. Stable and machine-readable; see contracts/openapi.yaml.
const (
	CodeValidationFailed  = "validation_failed"
	CodeUnauthenticated   = "unauthenticated"
	CodeNotFound          = "not_found"
	CodeImageTooLarge     = "image_too_large"
	CodeUnsupportedFormat = "unsupported_image_format"
	CodeUnsupportedMedia  = "unsupported_media_type"
	CodeInternal          = "internal_error"
)

// Field-level codes.
const (
	FieldRequired     = "required"
	FieldInvalidValue = "invalid_value"
	FieldNegative     = "negative_amount"
	FieldDateInFuture = "date_in_future"
	FieldTooLong      = "too_long"
	FieldUnknownImage = "unknown_image"
)

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type ErrorResponse struct {
	Error errorBody `json:"error"`
}

// WriteJSON renders a successful response.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The status and some bytes are already sent; there is nothing useful left to tell the
		// client. Record it so it is not silent.
		slog.Error("encoding response failed", "error", err)
	}
}

// WriteError renders the single error envelope every failing operation uses.
//
// The message is written for a collector to read. Internal detail — driver errors, SQL, stack
// traces — never reaches the client (Principle IV); it is logged instead.
func WriteError(w http.ResponseWriter, status int, code, message string, fields ...FieldError) {
	WriteJSON(w, status, ErrorResponse{Error: errorBody{Code: code, Message: message, Fields: fields}})
}

// WriteValidationFailed reports every problem found, together, in one response (FR-020).
func WriteValidationFailed(w http.ResponseWriter, fields []FieldError) {
	WriteError(w, http.StatusBadRequest, CodeValidationFailed,
		"This collectible could not be saved.", fields...)
}

// WriteUnauthenticated is the response when the acting collector cannot be resolved. No collection
// content accompanies it (FR-029).
func WriteUnauthenticated(w http.ResponseWriter) {
	WriteError(w, http.StatusUnauthorized, CodeUnauthenticated, "Sign in to view your collection.")
}

// WriteNotFound is returned both when a resource does not exist and when it belongs to another
// collector. The two are deliberately indistinguishable, so existence is never revealed (FR-027).
//
// This is why there is no 403 anywhere in this transport.
func WriteNotFound(w http.ResponseWriter) {
	WriteError(w, http.StatusNotFound, CodeNotFound, "Not found.")
}

// WriteInternal logs the cause and tells the client nothing about it.
func WriteInternal(w http.ResponseWriter, r *http.Request, cause error) {
	slog.ErrorContext(r.Context(), "request failed",
		"error", cause, "method", r.Method, "path", r.URL.Path)
	WriteError(w, http.StatusInternalServerError, CodeInternal,
		"Something went wrong. Please try again.")
}
