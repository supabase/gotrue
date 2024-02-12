package helpers

import (
	"github.com/gobuffalo/helpers/content"
	"github.com/gobuffalo/helpers/debug"
	"github.com/gobuffalo/helpers/encoders"
	"github.com/gobuffalo/helpers/env"
	"github.com/gobuffalo/helpers/escapes"
	"github.com/gobuffalo/helpers/forms"
	"github.com/gobuffalo/helpers/forms/bootstrap"
	"github.com/gobuffalo/helpers/hctx"
	"github.com/gobuffalo/helpers/inflections"
	"github.com/gobuffalo/helpers/iterators"
	"github.com/gobuffalo/helpers/meta"
	"github.com/gobuffalo/helpers/paths"
	"github.com/gobuffalo/helpers/tags"
	"github.com/gobuffalo/helpers/text"
)

var BootstrapForms = bootstrap.New()
var Content = content.New()
var Debug = debug.New()
var Encoders = encoders.New()
var Env = env.New()
var Escapes = escapes.New()
var Forms = forms.New()
var Inflections = inflections.New()
var Iterators = iterators.New()
var Meta = meta.New()
var Paths = paths.New()
var Tags = tags.New()
var Text = text.New()

var ALL = func() hctx.Map {
	return hctx.Merge(
		BootstrapForms,
		Content,
		Debug,
		Encoders,
		Env,
		Escapes,
		Inflections,
		Iterators,
		Meta,
		Paths,
		Tags,
		Text,
	)
}
