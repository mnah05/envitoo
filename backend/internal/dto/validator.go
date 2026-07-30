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
//
// Note 1: WithRequiredStructEnabled() was introduced in go-playground/validator
// v10.11+ to fix a subtle gotcha: without it, a struct field tagged `required`
// would not trigger a validation error if the field was a zero-value non-pointer
// struct (e.g. time.Time{}). Enabling this ensures the `required` tag behaves
// intuitively across all field types.
//
// Note 2: registerEventValidations attaches custom struct-level validators
// (like cross-field checks for date ranges). Keeping them in a separate
// function keeps this factory focused and testable.
func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	registerEventValidations(v)
	return v
}
