package api

import (
	"net/http"
)

//Authenticator describes the interface that a service authenticating an HTTP request should implement
type Authenticator interface {
	Authenticate(r *http.Request) (User, error)
}
