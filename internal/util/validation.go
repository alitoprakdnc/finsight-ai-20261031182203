package util

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ValidationError aggregates field‑level validation errors.
// It implements the error interface and can be inspected via the Errors map.
type ValidationError struct {
	Errors map[string]string `json:"errors"`
}

// Error implements the error interface.
func (ve ValidationError) Error() string {
	b, _ := json.Marshal(ve.Errors)
	return fmt.Sprintf("validation error: %s", string(b))
}

// IsValidationError reports whether err is a ValidationError.
func IsValidationError(err error) bool {
	_, ok := err.(ValidationError)
	return ok
}

// validatorSingleton holds a lazily‑initialized *validator.Validate instance.
var (
	validatorSingleton *validator.Validate
	once                sync.Once
)

// getValidator returns a singleton validator instance with common
// custom tags and JSON tag name extraction.
func getValidator() *validator.Validate {
	once.Do(func() {
		v := validator.New()

		// Use JSON tag name in error messages.
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})

		// Register custom validation functions.
		_ = v.RegisterValidation("uuid", validateUUID)
		_ = v.RegisterValidation("iso8601", validateISO8601)
		_ = v.RegisterValidation("enum", validateEnum)

		validatorSingleton = v
	})
	return validatorSingleton
}

// ValidateStruct validates a struct according to the tags defined on its fields.
// It returns a ValidationError containing a map of field → error message when validation fails.
func ValidateStruct(s interface{}) error {
	if s == nil {
		return ValidationError{Errors: map[string]string{"": "cannot validate nil"}}
	}
	v := getValidator()
	if err := v.Struct(s); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			fieldErrors := make(map[string]string, len(ve))
			for _, fe := range ve {
				field := fe.Field()
				// The field name is already the JSON tag thanks to RegisterTagNameFunc.
				msg := defaultErrorMessage(fe)
				fieldErrors[field] = msg
			}
			return ValidationError{Errors: fieldErrors}
		}
		// Non‑validation errors (e.g., bad struct) are returned as‑is.
		return err
	}
	return nil
}

// defaultErrorMessage builds a human‑readable message for a ValidationErrorField.
func defaultErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "iso8601":
		return "must be a valid ISO‑8601 datetime"
	case "enum":
		return fmt.Sprintf("must be one of [%s]", fe.Param())
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	default:
		return fmt.Sprintf("failed on '%s' validation", fe.Tag())
	}
}

// validateUUID checks that a field contains a valid UUID string.
func validateUUID(fl validator.FieldLevel) bool {
	val := fl.Field()
	if val.Kind() != reflect.String {
		return false
	}
	_, err := uuid.Parse(val.String())
	return err == nil
}

// validateISO8601 checks that a field contains a valid ISO‑8601 datetime.
// It accepts both date‑only and full datetime strings.
func validateISO8601(fl validator.FieldLevel) bool {
	val := fl.Field()
	if val.Kind() != reflect.String {
		return false
	}
	str := val.String()
	// Try a set of common ISO‑8601 layouts.
	layouts := []string{
		time.RFC3339,               // 2006-01-02T15:04:05Z07:00
		"2006-01-02",               // date only
		"2006-01-02T15:04:05",      // without timezone
		"2006-01-02 15:04:05",      // space separator
		"2006-01-02T15:04:05.999Z", // milliseconds
	}
	for _, l := range layouts {
		if _, err := time.Parse(l, str); err == nil {
			return true
		}
	}
	return false
}

// validateEnum checks that a field's value is one of the comma‑separated
// list provided in the tag parameter, e.g. `enum=admin,accountant,viewer`.
func validateEnum(fl validator.FieldLevel) bool {
	param := fl.Param()
	if param == "" {
		return false
	}
	allowed := strings.Split(param, ",")
	val := fl.Field()
	var value string

	switch val.Kind() {
	case reflect.String:
		value = val.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = fmt.Sprintf("%d", val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = fmt.Sprintf("%d", val.Uint())
	default:
		// Unsupported kind for enum validation.
		return false
	}

	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}