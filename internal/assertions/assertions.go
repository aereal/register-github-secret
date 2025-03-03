package assertions

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func DiffErrorsConservatively(want, got error) string {
	return cmp.Diff(want, got, cmpopts.EquateErrors())
}

func LiteralError(msg string) error { return literalError{Message: msg} }

type literalError struct {
	Message string
}

func (e literalError) Error() string { return e.Message }

func (e literalError) Is(other error) bool {
	if other == nil {
		return false
	}
	return e.Message == other.Error()
}
