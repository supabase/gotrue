package text

import "github.com/gobuffalo/helpers/hctx"

// Truncate will try to return a string that is no longer
// than `size`, which defaults to 50. If given
// a `trail` option the returned string will have
// that appended at the end, while still trying to make
// sure that the returned string is no longer than
// `size` characters long. However, if `trail` is longer
// than or equal to `size`, `trail` will be returned
// completely as is. Defaults to a `trail` of `...`.
func Truncate(s string, opts hctx.Map) string {
	if opts["size"] == nil {
		opts["size"] = 50
	}
	if opts["trail"] == nil {
		opts["trail"] = "..."
	}
	runesS := []rune(s)
	size := opts["size"].(int)
	if len(runesS) <= size {
		return s
	}
	trail := opts["trail"].(string)
	runesTrail := []rune(trail)
	if len(runesTrail) >= size {
		return trail
	}
	return string(runesS[:size-len(runesTrail)]) + trail
}
