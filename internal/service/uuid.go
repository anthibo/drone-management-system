package service

import "github.com/google/uuid"

func init() {
	uuidFunc = func() string { return uuid.NewString() }
}

