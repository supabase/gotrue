package bootstrap

import (
	"html/template"

	"github.com/gobuffalo/helpers/hctx"
	"github.com/gobuffalo/tags/v3"
	"github.com/gobuffalo/tags/v3/form/bootstrap"
)

// Form implements a `github.com/gobuffalo/plush` helper around the
// bootstrap.New function in the `github.com/gobuffalo/tags/form/bootstrap` package
func Form(opts tags.Options, help hctx.HelperContext) (template.HTML, error) {
	return helper(opts, help, func(opts tags.Options) helperable {
		return bootstrap.New(opts)
	})
}

// RemoteForm implements a `github.com/gobuffalo/plush` helper around the
// bootstrap.New function in the `github.com/gobuffalo/tags/form/bootstrap` package
func RemoteForm(opts tags.Options, help hctx.HelperContext) (template.HTML, error) {
	if opts == nil {
		opts = tags.Options{}
	}
	opts["data-remote"] = true
	return Form(opts, help)
}
