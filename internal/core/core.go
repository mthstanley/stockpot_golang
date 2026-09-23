package core

import "fmt"

type EntityNotFound struct {
	Type  string
	Ident string
	Err   error
}

func (e *EntityNotFound) Error() string {
	return fmt.Sprintf("%s entity with identifier %s not found", e.Type, e.Ident)
}

func (e *EntityNotFound) Unwrap() error {
	return e.Err
}
