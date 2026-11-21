package http

import (
	"encoding/json"
	"net/http"
)

// ErrorBody matches the canonical Fatman JSON error envelope.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ErrorEnvelope wraps an ErrorBody under the top-level "error" key.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// DataEnvelope wraps successful JSON payloads under the top-level "data" key.
type DataEnvelope struct {
	Data any `json:"data"`
}

// WriteJSON serializes a success payload wrapped in {"data": ...}.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(DataEnvelope{Data: data})
}

// WriteError serializes a structured JSON error envelope without leaking internals.
func WriteError(w http.ResponseWriter, status int, code, message string, details any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorEnvelope{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
