package iterators

// Iterator type can be implemented and used by the `for` command to build loops in templates
type Iterator interface {
	Next() interface{}
}
