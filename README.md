# grest - A full featured REST http handler in go

[![Godoc](https://godoc.org/github.com/xdbsoft/grest?status.png)](https://godoc.org/github.com/xdbsoft/grest)
[![Build Status](https://travis-ci.org/xdbsoft/grest.svg?branch=master)](https://travis-ci.org/xdbsoft/grest)
[![Coverage](http://gocover.io/_badge/github.com/xdbsoft/grest)](http://gocover.io/_badge/github.com/xdbsoft/grest)
[![Report](https://goreportcard.com/badge/github.com/xdbsoft/grest)](https://goreportcard.com/report/github.com/xdbsoft/grest)

## How-to

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

