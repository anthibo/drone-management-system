package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"penny-assesment/internal/domain"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal"
	message := "internal error"

	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		status = http.StatusUnauthorized
		code = "unauthorized"
		message = "unauthorized"
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		code = "forbidden"
		message = "forbidden"
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		message = "not found"
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		code = "conflict"
		message = "conflict"
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusUnprocessableEntity
		code = "invalid"
		message = "invalid request"
	case errors.Is(err, domain.ErrPrecondition):
		status = http.StatusConflict
		code = "precondition_failed"
		message = "precondition failed"
	case errors.Is(err, domain.ErrNoJob):
		status = http.StatusNotFound
		code = "no_job"
		message = "no job available"
	default:
		status = http.StatusInternalServerError
		code = "internal"
		message = "internal error"
	}
	respondJSON(w, status, errorResponse{Code: code, Message: message})
}
