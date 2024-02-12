package paths

import "github.com/gobuffalo/helpers/hctx"

// Keys to be used in templates for the functions in this package.
const (
	PathForKey = "pathFor"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		PathForKey: PathFor,
	}
}
