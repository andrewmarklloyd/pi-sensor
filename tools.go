// +build tools

package main

import (
	_ "github.com/andrewmarklloyd/do-app-firewall-entrypoint"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "golang.org/x/lint/golint"
)
