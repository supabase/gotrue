package tags

import (
	"html/template"

	"github.com/gobuffalo/tags/v3"
)

// CSS creates html for a css link using `github.com/gobuffalo/tags`
// to create a link with the given options and href. Defaults
// to `stylesheet` for `rel` and `screen` for `media` if those
// options are not set.
func CSS(href string, options tags.Options) template.HTML {
	if options["rel"] == nil {
		options["rel"] = "stylesheet"
	}

	if options["media"] == nil {
		options["media"] = "screen"
	}

	options["href"] = href
	cssTag := tags.New("link", options)

	return cssTag.HTML()
}
