//go:build !appengine
// +build !appengine

package pop

import (
	"fmt"

	"github.com/gobuffalo/fizz"
)

func newSchemaMigrations(name string) fizz.Table {
	tab := fizz.Table{
		Name: name,
		Columns: []fizz.Column{
			{
				Name:    "version",
				ColType: "string",
				Options: map[string]interface{}{
					"size": 14, // len(YYYYMMDDhhmmss)
				},
			},
		},
		Indexes: []fizz.Index{
			{Name: fmt.Sprintf("%s_version_idx", name), Columns: []string{"version"}, Unique: true},
		},
	}
	// this is for https://github.com/gobuffalo/pop/issues/659.
	// primary key is not necessary for the migration table but it looks like
	// some database engine versions requires it for index.
	tab.PrimaryKey("version")
	return tab
}
