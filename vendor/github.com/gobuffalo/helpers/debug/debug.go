package debug

import (
	"fmt"
	"html/template"

	"github.com/gobuffalo/helpers/hctx"
)

// Keys to be used in templates for the functions in this package.
const (
	DebugKey   = "debug"
	InspectKey = "inspect"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		DebugKey:   Debug,
		InspectKey: Inspect,
	}
}

// Debug by verbosely printing out using 'pre' tags.
func Debug(v interface{}) template.HTML {
	return template.HTML(fmt.Sprintf("<pre>%s</pre>", Inspect(v)))
}
