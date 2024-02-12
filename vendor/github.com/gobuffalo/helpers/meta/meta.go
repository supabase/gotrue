package meta

import (
	"github.com/gobuffalo/helpers/hctx"
)

// Keys to be used in templates for the functions in this package.
const (
	LenKey = "len"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		LenKey: Len,
	}
}
