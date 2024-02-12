package env

import (
	"fmt"
	"os"

	"github.com/gobuffalo/helpers/hctx"
)

// Keys to be used in templates for the functions in this package.
const (
	EnvKey   = "env"
	EnvOrKey = "envOr"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		EnvKey:   Env,
		EnvOrKey: EnvOr,
	}
}

// Env will return the specified environment variable,
// or an error if it can not be found
//	<%= env("GOPATH") %>
func Env(key string) (string, error) {
	s := os.Getenv(key)
	if len(s) == 0 {
		return "", fmt.Errorf("could not find ENV %q", key)
	}
	return s, nil
}

// Env will return the specified environment variable,
// or the second argument, if not found
//	<%= envOr("GOPATH", "~/go") %>
func EnvOr(key string, def string) string {
	s := os.Getenv(key)
	if len(s) == 0 {
		return def
	}
	return s
}
