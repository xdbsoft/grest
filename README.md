# grest - A full featured REST http handler in go

[![Godoc](https://godoc.org/github.com/xdbsoft/grest?status.png)](https://godoc.org/github.com/xdbsoft/grest)
[![Build Status](https://travis-ci.org/xdbsoft/grest.svg?branch=master)](https://travis-ci.org/xdbsoft/grest)
[![Coverage](http://gocover.io/_badge/github.com/xdbsoft/grest)](http://gocover.io/_badge/github.com/xdbsoft/grest)
[![Report](https://goreportcard.com/badge/github.com/xdbsoft/grest)](https://goreportcard.com/report/github.com/xdbsoft/grest)

## How-to

	package main

	import (
		"net/http"
		"github.com/xdbsoft/grest"
	)

	func main() {

		cfg := grest.Config{
			OpenIDConnectIssuer: "https://login.okiapps.com/", // You may use any OIDC provider (Google, Github, or self hosted)
			DBConnStr: "user=nestor password=nestor dbname=nestor sslmode=disable", //Connection string to the PostgreSQL database
			Collections: []api.CollectionDefinition{
				{
					Path: "test",
					Rules: []api.Rule{
						{
								Path: "test/{userId}/sub/{doc}",
								Allow: []api.Allow{
									{
										Methods: []api.Method{"READ"},
										If:      `path.doc != "private" || path.userId == user.id`,
									},
									{
										Methods: []api.Method{"WRITE","DELETE"},
										If:      `path.userId == user.id`,
									},
								},
						},
					}
				},
			},
		}
		
		http.Handle("/", grest.Server(cfg))
	}
