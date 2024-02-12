package pop

import (
	"fmt"
)

// Match holds the information parsed from a migration filename.
type Match struct {
	Version   string
	Name      string
	DBType    string
	Direction string
	Type      string
}

// ParseMigrationFilename parses a migration filename.
func ParseMigrationFilename(filename string) (*Match, error) {

	matches := mrx.FindAllStringSubmatch(filename, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	m := matches[0]

	var dbType string
	if m[3] == "" {
		dbType = "all"
	} else {
		dbType = CanonicalDialect(m[3][1:])
		if !DialectSupported(dbType) {
			return nil, fmt.Errorf("unsupported dialect %s", dbType)
		}
	}

	if m[5] == "fizz" && dbType != "all" {
		return nil, fmt.Errorf("invalid database type %q, expected \"all\" because fizz is database type independent", dbType)
	}

	match := &Match{
		Version:   m[1],
		Name:      m[2],
		DBType:    dbType,
		Direction: m[4],
		Type:      m[5],
	}

	return match, nil
}
