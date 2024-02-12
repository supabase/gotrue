package forms

import (
	"html/template"

	"github.com/gobuffalo/helpers/hctx"
	"github.com/gobuffalo/tags/v3"
	"github.com/gobuffalo/tags/v3/form"
)

// Form implements a `github.com/gobuffalo/plush` helper around the
// form.New function in the `github.com/gobuffalo/tags/form` package
func Form(opts tags.Options, help hctx.HelperContext) (template.HTML, error) {
	return helper(opts, help, func(opts tags.Options) helperable {
		return form.New(opts)
	})
}

// RemoteForm implements a `github.com/gobuffalo/plush` helper around the
// form.New function in the `github.com/gobuffalo/tags/form` package
func RemoteForm(opts tags.Options, help hctx.HelperContext) (template.HTML, error) {
	if opts == nil {
		opts = tags.Options{}
	}
	opts["data-remote"] = true
	return Form(opts, help)
}
