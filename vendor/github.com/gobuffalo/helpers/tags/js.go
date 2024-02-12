package tags

import (
	"html/template"

	"github.com/gobuffalo/tags/v3"
)

// JS creates html for a `<script>` tag using `github.com/gobuffalo/tags`
// to create the tag with the given `options` and `src`. Defaults
// to `text/javascript` for `type` if not set in the `options`.
func JS(src string, options tags.Options) template.HTML {
	if options["type"] == nil {
		options["type"] = "text/javascript"
	}

	options["src"] = src
	jsTag := tags.New("script", options)

	return jsTag.HTML()
}
