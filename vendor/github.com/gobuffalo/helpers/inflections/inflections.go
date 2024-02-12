package inflections

import (
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/gobuffalo/helpers/hctx"
)

// Keys to be used in templates for the functions in this package.
const (
	CamelizeKey    = "camelize"
	CapitalizeKey  = "capitalize"
	DasherizeKey   = "dasherize"
	OrdinalizeKey  = "ordinalize"
	PluralizeKey   = "pluralize"
	SingularizeKey = "singularize"
	UnderscoreKey  = "underscore"
	UpcaseKey      = "upcase"
	DowncaseKey    = "downcase"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		CamelizeKey:           Camelize,
		"camelize_down_first": Camelize, // Deprecated
		CapitalizeKey:         Capitalize,
		DasherizeKey:          Dasherize,
		OrdinalizeKey:         Ordinalize,
		PluralizeKey:          Pluralize,
		SingularizeKey:        Singularize,
		UnderscoreKey:         Underscore,
		DowncaseKey:           Downcase,
		UpcaseKey:             Upcase,

		// "asciffy":             Asciify,
		// "humanize":            Humanize,
		// "parameterize":        Parameterize,
		// "pluralize_with_size": PluralizeWithSize,
		// "tableize":            Tableize,
		// "typeify":             Typeify,
	}
}

var Upcase = strings.ToUpper
var Downcase = strings.ToLower
var Camelize = flect.Camelize
var Pascalize = flect.Pascalize
var Capitalize = flect.Capitalize
var Dasherize = flect.Dasherize
var Ordinalize = flect.Ordinalize
var Pluralize = flect.Pluralize
var Singularize = flect.Singularize
var Underscore = flect.Underscore
