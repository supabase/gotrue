package hctx

// Map is a standard map[string]interface{}
// for use throughout the helper packages.
type Map map[string]interface{}

// Merge creates a single Map from any
// number of Maps. Latter key/value pairs
// will overwrite earlier pairs.
func Merge(maps ...Map) Map {
	mx := map[string]interface{}{}
	for _, m := range maps {
		for k, v := range m {
			mx[k] = v
		}
	}
	return mx
}
