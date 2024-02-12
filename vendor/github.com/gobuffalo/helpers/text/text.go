package text

import "github.com/gobuffalo/helpers/hctx"

// Keys to be used in templates for the functions in this package.
const (
	MarkdownKey = "markdown"
	TruncateKey = "truncate"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		MarkdownKey: Markdown,
		TruncateKey: Truncate,
	}
}
