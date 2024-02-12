package translators

// MariaDB is a MariaDB-specific translator.
type MariaDB struct {
	*MySQL
}

// NewMariaDB constructs a new MariaDB translator.
func NewMariaDB(url, name string) *MariaDB {
	md := NewMySQL(url, name)
	md.strDefaultSize = 191
	return &MariaDB{
		MySQL: md,
	}
}

func (MariaDB) Name() string {
	return "mariadb"
}
