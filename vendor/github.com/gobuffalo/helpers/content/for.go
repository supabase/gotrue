package content

import (
	"html/template"

	"github.com/gobuffalo/helpers/hctx"
)

// ContentFor stores a block of templating code to be re-used later in the template
// via the contentOf helper.
// An optional map of values can be passed to contentOf,
// which are made available to the contentFor block.
/*
	<% contentFor("buttons") { %>
		<button>hi</button>
	<% } %>
*/
func ContentFor(name string, help hctx.HelperContext) {
	help.Set("contentFor:"+name, func(data hctx.Map) (template.HTML, error) {
		hctx := help.New()
		for k, v := range data {
			hctx.Set(k, v)
		}
		body, err := help.BlockWith(hctx)
		if err != nil {
			return "", err
		}
		return template.HTML(body), nil
	})
}
