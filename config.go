package grest

import (
	"github.com/xdbsoft/grest/rules"
)

// Config contains all required information for the intialisation of a grest server
type Config struct {
	OpenIDConnectIssuer string
	DBConnStr           string
	Rules               []rules.Rule
}
