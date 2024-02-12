package content

import "github.com/gobuffalo/helpers/hctx"

// Keys to be used in templates for the functions in this package.
const (
	OfKey      = "contentOf"
	ForKey     = "contentFor"
	DefaultKey = "default"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		OfKey:      ContentOf,
		ForKey:     ContentFor,
		DefaultKey: WithDefault,
	}
}
