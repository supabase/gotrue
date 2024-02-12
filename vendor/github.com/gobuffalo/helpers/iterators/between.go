package iterators

// Between will iterate up to, but not including `b`
//	Between(0,10) // 0,1,2,3,4,5,6,7,8,9
func Between(a, b int) Iterator {
	return &ranger{pos: a, end: b - 1}
}
