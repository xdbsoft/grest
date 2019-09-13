package grest

import (
	"github.com/xdbsoft/grest/rules"
)

// CollectionDefinition describes the structure and the authorization for a collection
type CollectionDefinition struct {
	Name  string
	Rules []rules.Rule
}

// Config contains all required information for the intialisation of a grest server
type Config struct {
	OpenIDConnectIssuer string
	DBConnStr           string
	Collections         []CollectionDefinition
}
