package form

import "github.com/gobuffalo/tags/v3"

//SubmitTag generates an input tag with type "submit"
func (f Form) SubmitTag(value string, opts tags.Options) *tags.Tag {
	opts["type"] = "submit"
	opts["value"] = value
	return tags.New("input", opts)
}
