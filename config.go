package grest

import (
	"github.com/xdbsoft/grest/api"
)

// Config contains all required information for the intialisation of a grest server
type Config struct {
	OpenIDConnectIssuer string
	DBConnStr           string
	Collections         []api.CollectionDefinition
}
