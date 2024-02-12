package name

import (
	"regexp"
	"strings"

	"github.com/gobuffalo/flect"
)

var alphanum = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// Folder returns a suitable folder name. It removes any special characters
// from the given string `s` and returns a string consists of alpha-numeric
// characters.
//	admin/widget --> admin/widget
//	adminUser --> admin_user
//	foo_bar --> foo_bar
//	Admin --> admin
//	U$ser --> u_ser
func Folder(s string, exts ...string) string {
	return New(s).Folder(exts...).String()
}

// Folder returns a suitable folder name. It removes any special characters
// from the given string `s` and returns a string consists of alpha-numeric
// characters.
//	admin/widget --> admin/widget
//	adminUser --> admin_user
//	foo_bar --> foo_bar
//	Admin --> admin
//	U$ser --> u_ser
func (i Ident) Folder(exts ...string) Ident {
	var parts []string

	for _, part := range strings.Split(i.Original, "/") {
		part = alphanum.ReplaceAllString(flect.Underscore(part), "")
		parts = append(parts, part)
	}

	return New(strings.Join(parts, "/") + strings.Join(exts, ""))
}
