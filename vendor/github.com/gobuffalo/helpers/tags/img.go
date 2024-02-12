package tags

import (
	"html/template"

	"github.com/gobuffalo/tags/v3"
)

// Img creates html for a `<img>` tag using `github.com/gobuffalo/tags`
// to create a tag with the given `src` and `options`.
func Img(src string, options tags.Options) template.HTML {
	options["src"] = src
	imgTag := tags.New("img", options)

	return imgTag.HTML()
}
