package iterators

// Until will iterate up to, but not including `a`
//	Until(3) // 0,1,2
func Until(a int) Iterator {
	return &ranger{pos: -1, end: a - 1}
}
