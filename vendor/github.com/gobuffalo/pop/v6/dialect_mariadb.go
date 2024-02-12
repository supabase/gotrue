package pop

import (
	"github.com/gobuffalo/fizz"
	"github.com/gobuffalo/fizz/translators"
)

const nameMariaDB = "mariadb"

func init() {
	AvailableDialects = append(AvailableDialects, nameMariaDB)
	urlParser[nameMariaDB] = urlParserMySQL
	finalizer[nameMariaDB] = finalizerMySQL
	newConnection[nameMariaDB] = newMySQL
}

var _ dialect = &mariaDB{}

type mariaDB struct {
	mysql
}

func (m *mariaDB) Name() string {
	return nameMariaDB
}

func (m *mariaDB) FizzTranslator() fizz.Translator {
	t := translators.NewMariaDB(m.URL(), m.Details().Database)
	return t
}
