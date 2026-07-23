package http

import "fmt"

// BindingError reports the field and input value that failed conversion.
type BindingError struct {
	Field string
	Value string
	Err   error
}

func (e *BindingError) Error() string {
	return fmt.Sprintf("bind field %q with value %q: %v", e.Field, e.Value, e.Err)
}

func (e *BindingError) Unwrap() error {
	return e.Err
}
