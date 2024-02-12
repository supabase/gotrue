package pop

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"text/template"

	"github.com/gobuffalo/fizz"
)

// MigrationContent returns the content of a migration.
func MigrationContent(mf Migration, c *Connection, r io.Reader, usingTemplate bool) (string, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil
	}

	content := ""
	if usingTemplate {
		t := template.Must(template.New("migration").Parse(string(b)))
		var bb bytes.Buffer
		err = t.Execute(&bb, c.Dialect.Details())
		if err != nil {
			return "", fmt.Errorf("could not execute migration template %s: %w", mf.Path, err)
		}
		content = bb.String()
	} else {
		content = string(b)
	}

	if mf.Type == "fizz" {
		content, err = fizz.AString(content, c.Dialect.FizzTranslator())
		if err != nil {
			return "", fmt.Errorf("could not fizz the migration %s: %w", mf.Path, err)
		}
	}

	return content, nil
}
