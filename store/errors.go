package store

import (
	"errors"
	"fmt"
)

var ErrInvalidQueryOptions = errors.New("invalid query options")
var ErrInvalidExecOptions = errors.New("invalid exec options")
var ErrInvalidTableCreationOptions = errors.New("invalid table creation options")

func NewInvalidQueryOptions(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidQueryOptions, msg)
}

func NewInvalidExecOptions(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidExecOptions, msg)
}

func NewInvalidTableCreationOptions(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidTableCreationOptions, msg)
}
