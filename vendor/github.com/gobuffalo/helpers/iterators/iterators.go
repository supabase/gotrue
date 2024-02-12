package iterators

import "github.com/gobuffalo/helpers/hctx"

// Keys to be used in templates for the functions in this package.
const (
	RangeKey   = "range"
	BetweenKey = "between"
	UntilKey   = "until"
	GroupByKey = "groupBy"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		RangeKey:   Range,
		BetweenKey: Between,
		UntilKey:   Until,
		GroupByKey: GroupBy,
	}
}
