package content

import (
	"errors"
	"html/template"

	"github.com/gobuffalo/helpers/hctx"
)

// ContentOf retrieves a stored block for templating and renders it.
// You can pass an optional map of fields that will be set.
/*
	<%= contentOf("buttons") %>
	<%= contentOf("buttons", {"label": "Click me"}) %>
*/
func ContentOf(name string, data hctx.Map, help hctx.HelperContext) (template.HTML, error) {
	fn, ok := help.Value("contentFor:" + name).(func(data hctx.Map) (template.HTML, error))
	if !ok {
		if !help.HasBlock() {
			return template.HTML(""), errors.New("missing contentOf block: " + name)
		}

		hc := help.New()
		for k, v := range data {
			hc.Set(k, v)
		}
		body, err := help.BlockWith(hc)
		if err != nil {
			return template.HTML(""), err
		}

		return template.HTML(body), nil
	}
	return fn(data)
}
