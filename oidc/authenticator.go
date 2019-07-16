package oidc

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/xdbsoft/grest/api"
)

func New(openIDConnectIssuer string) (api.Authenticator, error) {

	provider, err := oidc.NewProvider(context.Background(), openIDConnectIssuer)
	if err != nil {
		return nil, err
	}

	config := oidc.Config{
		SkipClientIDCheck: true,
	}

	verifier := provider.Verifier(&config)

	return &authenticator{
		Verifier: verifier,
	}, nil
}

type authenticator struct {
	Verifier *oidc.IDTokenVerifier
}

//getRawIDToken returns the raw token if any
func getRawIDToken(r *http.Request) string {

	//Retrieve JWT from Authorization header (or auth form parameter)
	bearerString := r.Header.Get("Authorization")
	if len(bearerString) == 0 {
		formBearer := r.FormValue("auth")
		if len(formBearer) > 0 {
			bearerString = "Bearer " + formBearer
		}
	}
	if len(bearerString) < len("Bearer ") {
		return ""
	}
	return bearerString[len("Bearer "):]
}

func (a *authenticator) Authenticate(r *http.Request) (api.User, error) {

	ctx := r.Context()

	rawIDToken := getRawIDToken(r)
	if len(rawIDToken) == 0 {
		return api.User{}, nil
	}

	idToken, err := a.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Println(err)
		return api.User{}, notAuthorizedError{}
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return api.User{}, err
	}

	return api.User{
		ID:    idToken.Subject,
		Name:  claims.Name,
		Email: claims.Email,
	}, nil
}

type notAuthorizedError struct {
}

func (err notAuthorizedError) Error() string {
	return fmt.Sprintf("Invalid credential")
}

func (err notAuthorizedError) IsNotAuthorized() bool {
	return true
}
