package main

import "fmt"

type CRUD[T any, I any] interface {
	Create(T) (I, error)
	ReadOne(I) (T, error)
	ReadMany() ([]T, error)
	Update(T) error
	Delete(I) error
}

type DuplicateError struct {
	Name   string
	Reason string
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("Cannot perform operation on %v because it is considered duplication due to: %v",
		e.Name, e.Reason)
}
