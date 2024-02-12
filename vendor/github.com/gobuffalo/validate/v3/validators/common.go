package validators

import (
	"github.com/gobuffalo/flect"
)

var CustomKeys = map[string]string{}

func GenerateKey(s string) string {
	key := CustomKeys[s]
	if key != "" {
		return key
	}
	return flect.Underscore(s)
}
