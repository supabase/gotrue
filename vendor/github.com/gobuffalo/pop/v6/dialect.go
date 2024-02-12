package pop

import (
	"io"

	"github.com/gobuffalo/fizz"
	"github.com/gobuffalo/pop/v6/columns"
)

type crudable interface {
	SelectOne(*Connection, *Model, Query) error
	SelectMany(*Connection, *Model, Query) error
	Create(*Connection, *Model, columns.Columns) error
	Update(*Connection, *Model, columns.Columns) error
	UpdateQuery(*Connection, *Model, columns.Columns, Query) (int64, error)
	Destroy(*Connection, *Model) error
	Delete(*Connection, *Model, Query) error
}

type fizzable interface {
	FizzTranslator() fizz.Translator
}

type quotable interface {
	Quote(key string) string
}

type dialect interface {
	crudable
	fizzable
	quotable
	Name() string
	DefaultDriver() string
	URL() string
	MigrationURL() string
	Details() *ConnectionDetails
	TranslateSQL(string) string
	CreateDB() error
	DropDB() error
	DumpSchema(io.Writer) error
	LoadSchema(io.Reader) error
	Lock(func() error) error
	TruncateAll(*Connection) error
}

type afterOpenable interface {
	AfterOpen(*Connection) error
}
