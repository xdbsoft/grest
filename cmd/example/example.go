package main

import (
	"log"
	"net/http"

	"github.com/xdbsoft/grest"
	"github.com/xdbsoft/grest/rules"
)

func main() {

	cfg := grest.Config{
		OpenIDConnectIssuer: "https://login.okiapps.com/",                                // You may use any OIDC provider (Google, Github, or self hosted)
		DBConnStr:           "user=nestor password=nestor dbname=nestor sslmode=disable", //Connection string to the PostgreSQL database
		Rules: []rules.Rule{
			{
				Path: "test/{userId}/sub/{doc}",
				Read: rules.Allow{
					With: []rules.With{
						{
							Name: "user",
							Path: "test/{userId}",
						},
					},
					IfPath: `path.doc != "private" || path.userId == user.id || with.user.role == "admin"`,
				},
				Write: rules.Allow{
					IfPath:    `path.userId == user.id`,
					IfContent: `content.properties.policy == 'EDITABLE' || with.user.role == "admin"`,
				},
			},
		},
	}

	s, _ := grest.Server(cfg)

	http.Handle("/", s)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
