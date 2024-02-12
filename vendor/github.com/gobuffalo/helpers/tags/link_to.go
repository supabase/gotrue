package tags

import (
	"errors"
	"html/template"

	"github.com/gobuffalo/helpers/hctx"
	"github.com/gobuffalo/helpers/paths"
	"github.com/gobuffalo/tags/v3"
)

// LinkTo creates html for a `<a>` tag using `github.com/gobuffalo/tags`
// to create tag with the given options(`opts`) and
// using [`paths#PathFor(in)`](https://godoc.org/github.com/gobuffalo/helpers/paths#PathFor) to set the `href`. If given a block
// it will be interrupted and appended inside of the `<a>` tag.
// Example 1:
//	<%= linkTo([user, widget], {class: "btn"}) %>
// yields(assuming user ID'd by `id` and widget by `slug`):
//	<a class="btn" href="/users/id/widget/slug"></a>
// Example 2:
//	<%= linkTo("foo", {class: "btn", body: "View"}) %>
// yields:
//	<a class="btn" href="/foo">View</a>
func LinkTo(in interface{}, opts tags.Options, help hctx.HelperContext) (template.HTML, error) {
	s, err := paths.PathFor(in)
	if err != nil {
		return "", err
	}

	if opts == nil {
		opts = tags.Options{}
	}

	if help == nil {
		return "", errors.New("nil HelperContext")
	}

	opts["href"] = s
	a := tags.New("a", opts)
	if help.HasBlock() {
		x, err := help.Block()
		if err != nil {
			return "", err
		}
		a.Append(x)
	}

	return a.HTML(), nil
}

// RemoteLinkTo creates an AJAXified `<a>` tag.
//	<%= remoteLinkTo(widget, {class: "btn btn-info", body: "View"}) %>
//	<a class="btn btn-info" data-remote="true" href="/widgets/b6b0ab24-19ae-4cdd-ad73-c5ecbddd6f91">View</a>
func RemoteLinkTo(in interface{}, opts tags.Options, help hctx.HelperContext) (template.HTML, error) {
	if opts == nil {
		opts = tags.Options{}
	}

	opts["data-remote"] = true

	return LinkTo(in, opts, help)
}
