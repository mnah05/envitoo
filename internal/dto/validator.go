package dto

import (
	"github.com/go-playground/validator/v10"
)

// NewValidator returns a new validator.Validate instance configured with
// required-struct mode enabled (so `required` on non-pointer fields
// works as expected) and all custom struct-level validations registered.
//
// Use this to construct a validator for injection into the server or
// services. Each call returns an independent instance — safe to
// customize further without affecting other callers.
func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	registerEventValidations(v)
	return v
}
