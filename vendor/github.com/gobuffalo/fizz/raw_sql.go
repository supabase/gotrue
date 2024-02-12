package fizz

import (
	"strings"
)

func (f fizzer) RawSQL(sql string) error {
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}
	return f.add(sql, nil)
}

// Deprecated: use RawSQL instead.
func (f fizzer) RawSql(sql string) error {
	return f.RawSQL(sql)
}
