package hctx

import (
	"context"
)

type Context interface {
	context.Context
	New() Context
	Has(key string) bool
	Set(key string, value interface{})
}

type HelperContext interface {
	Context
	Block() (string, error)
	BlockWith(Context) (string, error)
	HasBlock() bool
	Render(s string) (string, error)
}
