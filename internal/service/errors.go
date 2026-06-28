package service

import "errors"

var (
	ErrEventNotFound        = errors.New("event not found")
	ErrRegistrationNotFound = errors.New("registration not found")
	ErrInvalidEvent         = errors.New("invalid event")
	ErrConflict             = errors.New("registration already exists")
	ErrEventFull            = errors.New("event is full")
	ErrEventNotOpen         = errors.New("event not open for registration")
)
