package cli

import (
	"errors"
	"fmt"
)

type MissingTokenError struct{}

func (MissingTokenError) Error() string { return "missing GitHub token" }

var ErrMissingToken MissingTokenError

type SecretNameRequiredError struct{}

func (SecretNameRequiredError) Error() string { return "secret name required" }

var ErrSecretNameRequired SecretNameRequiredError

type SecretValueRequiredError struct{}

func (SecretValueRequiredError) Error() string { return "secret value required" }

var ErrSecretValueRequired SecretValueRequiredError

type MalformedQualifiedRepoError struct {
	Input string
}

func (e *MalformedQualifiedRepoError) Error() string {
	return fmt.Sprintf("malformed qualified repository name: %q", e.Input)
}

func (e *MalformedQualifiedRepoError) Is(err error) bool {
	thatErr := new(MalformedQualifiedRepoError)
	if !errors.As(err, &thatErr) {
		return false
	}
	return e.Input == thatErr.Input
}
