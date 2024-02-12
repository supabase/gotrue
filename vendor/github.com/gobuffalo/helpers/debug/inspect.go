package debug

import (
	"fmt"
)

// Inspect the interface using the `%+v` formatter
func Inspect(v interface{}) string {
	return fmt.Sprintf("%+v", v)
}
