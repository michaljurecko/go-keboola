package schema

import (
	"fmt"

	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

type ValidationError struct {
	message string
}

type FieldValidationError struct {
	path    string
	message string
}

func (v ValidationError) Error() string {
	return v.message
}

func (v ValidationError) WriteError(w errors.Writer, _ int, _ errors.StackTrace) {
	// Disable other formatting
	w.Write(v.Error())
}

// Path to the JSON field.
func (v FieldValidationError) Path() string {
	return v.path
}

func (v FieldValidationError) Error() string {
	return fmt.Sprintf(`"%s": %s`, v.path, v.message)
}

func (v FieldValidationError) WriteError(w errors.Writer, _ int, _ errors.StackTrace) {
	// Disable other formatting
	w.Write(v.Error())
}
