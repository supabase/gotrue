package encoders

import "html/template"

// Raw converts a `string` to a `template.HTML`
func Raw(s string) template.HTML {
	return template.HTML(s)
}
