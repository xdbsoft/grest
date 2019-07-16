package grest

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/xdbsoft/grest/api"
)

//IsNotFound returns whether the error cause is that something was not found
func IsNotFound(err error) bool {
	nfe, ok := errors.Cause(err).(NotFound)
	return ok && nfe.IsNotFound()
}

//NotFound is the interface that wraps the IsNotFound nethod
type NotFound interface {
	IsNotFound() bool
}

//IsNotAuthorized returns whether the error cause is that there was an attempt to perform a not authorized action
func IsNotAuthorized(err error) bool {
	nae, ok := errors.Cause(err).(NotAuthorized)
	return ok && nae.IsNotAuthorized()
}

//NotAuthorized is the interface that wraps the IsNotAuthorized nethod
type NotAuthorized interface {
	IsNotAuthorized() bool
}

//IsBadRequest returns whether the error cause is that the provided inputs are incorrect
func IsBadRequest(err error) bool {
	nae, ok := errors.Cause(err).(BadRequest)
	return ok && nae.IsBadRequest()
}

//BadRequest is the interface that wraps the IsBadRequest method
type BadRequest interface {
	IsBadRequest() bool
}

type badRequest string

func (err badRequest) IsBadRequest() bool {
	return true
}
func (err badRequest) Error() string {
	return string(err)
}

type notAuthorizedError struct {
	Target api.ObjectRef
}

func (err notAuthorizedError) Error() string {
	return fmt.Sprintf("Not authorized to access '%s'", err.Target)
}

func (err notAuthorizedError) IsNotAuthorized() bool {
	return true
}

type notFoundError struct {
	Target api.ObjectRef
}

func (err notFoundError) Error() string {
	return fmt.Sprintf("Target not found: '%s'", err.Target)
}

func (err notFoundError) IsNotFound() bool {
	return true
}
