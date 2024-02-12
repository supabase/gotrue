package escapes

import (
	"html/template"

	"github.com/gobuffalo/helpers/hctx"
)

// HTMLEscape will escape a string for HTML
func HTMLEscape(s string, help hctx.HelperContext) (string, error) {
	var err error
	if help.HasBlock() {
		s, err = help.Block()
	}
	if err != nil {
		return "", err
	}
	return template.HTMLEscapeString(s), nil
}
