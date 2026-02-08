package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalid       = errors.New("invalid")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrNoJob         = errors.New("no job available")
	ErrPrecondition  = errors.New("precondition failed")
)

