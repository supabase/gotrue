package iterators

// Range creates an Iterator that will
// iterate numbers from a to b, including b.
func Range(a, b int) Iterator {
	return &ranger{pos: a - 1, end: b}
}

type ranger struct {
	pos int
	end int
}

// Next returns the next number in the Range or nil
func (r *ranger) Next() interface{} {
	if r.pos < r.end {
		r.pos++
		return r.pos
	}
	return nil
}
