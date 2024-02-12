package translators

import (
	"fmt"
	"strings"

	"github.com/gobuffalo/fizz"
)

type SQLite struct {
	Schema SchemaQuery
}

func NewSQLite(url string) *SQLite {
	schema := &sqliteSchema{
		Schema{
			URL:    url,
			schema: map[string]*fizz.Table{},
		},
	}
	schema.Builder = schema
	return &SQLite{
		Schema: schema,
	}
}

func (SQLite) Name() string {
	return "sqlite3"
}

func (p *SQLite) CreateTable(t fizz.Table) (string, error) {
	p.Schema.SetTable(&t)

	sql := []string{}
	cols := []string{}
	var s string
	for _, c := range t.Columns {
		if c.Primary {
			switch strings.ToLower(c.ColType) {
			case "integer", "int":
				s = fmt.Sprintf("\"%s\" INTEGER PRIMARY KEY AUTOINCREMENT", c.Name)
			case "string", "text", "uuid":
				s = fmt.Sprintf("\"%s\" TEXT PRIMARY KEY", c.Name)
			default:
				return "", fmt.Errorf("can not use %s as a primary key", c.ColType)
			}
		} else {
			s = p.buildColumn(c)
		}
		cols = append(cols, s)
	}

	for _, fk := range t.ForeignKeys {
		cols = append(cols, p.buildForeignKey(t, fk, true))
	}

	primaryKeys := t.PrimaryKeys()
	if len(primaryKeys) > 1 {
		pks := make([]string, len(primaryKeys))
		for i, pk := range primaryKeys {
			pks[i] = fmt.Sprintf("\"%s\"", pk)
		}
		cols = append(cols, fmt.Sprintf("PRIMARY KEY(%s)", strings.Join(pks, ", ")))
	}

	s = fmt.Sprintf("CREATE TABLE \"%s\" (\n%s\n);", t.Name, strings.Join(cols, ",\n"))
	sql = append(sql, s)

	for _, i := range t.Indexes {
		s, err := p.AddIndex(fizz.Table{
			Name:    t.Name,
			Indexes: []fizz.Index{i},
		})
		if err != nil {
			return "", err
		}
		sql = append(sql, s)
	}
	return strings.Join(sql, "\n"), nil
}

func (p *SQLite) DropTable(t fizz.Table) (string, error) {
	p.Schema.Delete(t.Name)
	s := fmt.Sprintf("DROP TABLE \"%s\";", t.Name)
	return s, nil
}

func (p *SQLite) RenameTable(t []fizz.Table) (string, error) {
	if len(t) < 2 {
		return "", fmt.Errorf("not enough table names supplied")
	}
	oldName := t[0].Name
	newName := t[1].Name
	tableInfo, err := p.Schema.TableInfo(oldName)
	if err != nil {
		return "", err
	}
	tableInfo.Name = newName
	s := fmt.Sprintf("ALTER TABLE \"%s\" RENAME TO \"%s\";", oldName, newName)
	return s, nil
}

func (p *SQLite) ChangeColumn(t fizz.Table) (string, error) {
	tableInfo, err := p.Schema.TableInfo(t.Name)

	if err != nil {
		return "", err
	}

	for i := range tableInfo.Columns {
		if tableInfo.Columns[i].Name == t.Columns[0].Name {
			tableInfo.Columns[i] = t.Columns[0]
			break
		}
	}

	sql := []string{}

	var copyIndexes = make([]fizz.Index, len(tableInfo.Indexes))
	for k, i := range tableInfo.Indexes {
		s, err := p.DropIndex(fizz.Table{
			Name:    tableInfo.Name,
			Indexes: []fizz.Index{i},
		})
		if err != nil {
			return "", err
		}
		sql = append(sql, s)
		copyIndexes[k] = i
	}
	tableInfo.Indexes = copyIndexes // We need to recreate those

	// We do not need to use withForeignKeyPreservingTempTable here because this will not touch any foreign keys!
	s, err := p.withForeignKeyPreservingTempTable(*tableInfo, t.Name, func(newTable fizz.Table, tableName string) (string, error) {
		if t.Columns[0].Name == "slug" {
			fmt.Print("asdf")
		}
		return fmt.Sprintf("INSERT INTO \"%s\" (%s) SELECT %s FROM \"%s\";", newTable.Name, strings.Join(newTable.ColumnNames(), ", "), strings.Join(newTable.ColumnNames(), ", "), tableName), nil
	})

	if err != nil {
		return "", err
	}

	sql = append(sql, s)

	return strings.Join(sql, "\n"), nil
}

func (p *SQLite) AddColumn(t fizz.Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	c := t.Columns[0]

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}

	tableInfo.Columns = append(tableInfo.Columns, c)

	s := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN %s;", t.Name, p.buildColumn(c))
	return s, nil
}

func (p *SQLite) DropColumn(t fizz.Table) (string, error) {
	if len(t.Columns) < 1 {
		return "", fmt.Errorf("not enough columns supplied")
	}

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}

	sql := []string{}
	droppedColumn := t.Columns[0]

	newColumns := []fizz.Column{}
	for _, c := range tableInfo.Columns {
		if c.Name != droppedColumn.Name {
			newColumns = append(newColumns, c)
		}
	}
	tableInfo.Columns = newColumns

	newIndexes := []fizz.Index{}
	for _, i := range tableInfo.Indexes {
		s, err := p.DropIndex(fizz.Table{
			Name:    tableInfo.Name,
			Indexes: []fizz.Index{i},
		})
		if err != nil {
			return "", err
		}
		sql = append(sql, s)
		if tableInfo.HasColumns(i.Columns...) {
			newIndexes = append(newIndexes, i)
		}
	}
	tableInfo.Indexes = newIndexes

	newForeignKeys := []fizz.ForeignKey{}
	for _, i := range tableInfo.ForeignKeys {
		if tableInfo.HasColumns(i.Column) {
			newForeignKeys = append(newForeignKeys, i)
		}
	}
	tableInfo.ForeignKeys = newForeignKeys

	s, err := p.withForeignKeyPreservingTempTable(*tableInfo, t.Name, func(newTable fizz.Table, tableName string) (string, error) {
		return fmt.Sprintf("INSERT INTO \"%s\" (%s) SELECT %s FROM \"%s\";\n", newTable.Name, strings.Join(newTable.ColumnNames(), ", "), strings.Join(newTable.ColumnNames(), ", "), tableName), nil
	})

	if err != nil {
		return "", err
	}
	sql = append(sql, s)

	return strings.Join(sql, "\n"), nil
}

func (p *SQLite) RenameColumn(t fizz.Table) (string, error) {
	if len(t.Columns) < 2 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	oc := t.Columns[0]
	nc := t.Columns[1]
	s := fmt.Sprintf("ALTER TABLE \"%s\" RENAME COLUMN \"%s\" TO \"%s\";", t.Name, oc.Name, nc.Name)
	return s, nil
}

func (p *SQLite) AddIndex(t fizz.Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", fmt.Errorf("not enough indexes supplied")
	}
	i := t.Indexes[0]
	s := fmt.Sprintf("CREATE INDEX \"%s\" ON \"%s\" (%s);", i.Name, t.Name, strings.Join(i.Columns, ", "))
	if i.Unique {
		s = strings.Replace(s, "CREATE", "CREATE UNIQUE", 1)
	}

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}
	tableInfo.Indexes = append(tableInfo.Indexes, i)
	return s, nil
}

func (p *SQLite) DropIndex(t fizz.Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", fmt.Errorf("not enough indexes supplied")
	}
	i := t.Indexes[0]
	s := fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";", i.Name)

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}
	newIndexes := []fizz.Index{}
	for _, c := range tableInfo.Indexes {
		if c.Name != i.Name {
			newIndexes = append(newIndexes, c)
		}
	}
	tableInfo.Indexes = newIndexes

	return s, nil
}

func (p *SQLite) RenameIndex(t fizz.Table) (string, error) {
	if len(t.Indexes) < 2 {
		return "", fmt.Errorf("not enough indexes supplied")
	}

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}

	sql := []string{}

	oldIndex := t.Indexes[0]
	newIndex := t.Indexes[1]

	for _, ti := range tableInfo.Indexes {
		if ti.Name == oldIndex.Name {
			ti.Name = newIndex.Name
			newIndex = ti
			break
		}
	}

	s, err := p.DropIndex(fizz.Table{
		Name:    tableInfo.Name,
		Indexes: []fizz.Index{oldIndex},
	})

	if err != nil {
		return "", err
	}

	sql = append(sql, s)

	s, err = p.AddIndex(fizz.Table{
		Name:    t.Name,
		Indexes: []fizz.Index{newIndex},
	})

	if err != nil {
		return "", err
	}

	sql = append(sql, s)

	return strings.Join(sql, "\n"), nil
}

func (p *SQLite) AddForeignKey(t fizz.Table) (string, error) {
	return "", fmt.Errorf("SQLite does not support this feature")
}

func (p *SQLite) DropForeignKey(t fizz.Table) (string, error) {
	return "", fmt.Errorf("SQLite does not support this feature")
}

func (p *SQLite) withTempTable(table string, fn func(fizz.Table) (string, error)) (string, error) {
	tempTable := fizz.Table{Name: fmt.Sprintf("_%s_tmp", table)}

	s, err := fn(tempTable)
	if err != nil {
		return "", err
	}

	sql := []string{
		fmt.Sprintf("ALTER TABLE \"%s\" RENAME TO \"%s\";", table, tempTable.Name),
	}

	sql = append(sql, s, fmt.Sprintf("DROP TABLE \"%s\";", tempTable.Name))

	return strings.Join(sql, "\n"), nil
}

// withForeignKeyPreservingTempTable create a new temporary table, copies all the contents from the old table over to the new
// table, removes the old table, and then renames the temporary table to the original table name. This
// preserves foreign key constraint because because SQLite does not drop foreign keys when their reference table
// is deleted. It only removes any columns referencing data in the deleted table. [1]
//
// [1] https://sqlite.org/lang_droptable.html
func (p *SQLite) withForeignKeyPreservingTempTable(newTable fizz.Table, tableName string, fn func(newTable fizz.Table, tableName string) (string, error)) (string, error) {
	var sql []string

	newTable.Name = fmt.Sprintf("_%s_tmp", tableName)
	defer func() {
		newTable.Name = tableName
	}()

	createTableSQL, err := p.CreateTable(newTable)
	if err != nil {
		return "", err
	}

	callbackSQL, err := fn(newTable, tableName)
	if err != nil {
		return "", err
	}

	return strings.Join(append(sql,
		createTableSQL,
		callbackSQL,
		fmt.Sprintf("DROP TABLE \"%s\";", tableName),
		fmt.Sprintf("ALTER TABLE \"%s\" RENAME TO \"%s\";", newTable.Name, tableName),
	), "\n"), nil
}

func (p *SQLite) buildColumn(c fizz.Column) string {
	s := fmt.Sprintf("\"%s\" %s", c.Name, p.colType(c))
	if c.Options["null"] == nil {
		s = fmt.Sprintf("%s NOT NULL", s)
	}
	if c.Options["default"] != nil {
		s = fmt.Sprintf("%s DEFAULT '%v'", s, c.Options["default"])
	}
	if c.Options["default_raw"] != nil {
		s = fmt.Sprintf("%s DEFAULT %s", s, c.Options["default_raw"])
	}
	return s
}

func (p *SQLite) colType(c fizz.Column) string {
	switch strings.ToLower(c.ColType) {
	case "uuid":
		return "char(36)"
	case "timestamp", "time", "datetime":
		return "DATETIME"
	case "boolean", "date":
		return "NUMERIC"
	case "string", "text":
		return "TEXT"
	case "int", "integer":
		return "INTEGER"
	case "float":
		// precision and scale not supported here
		return "REAL"
	case "json":
		return "TEXT"
	case "blob", "[]byte":
		return "BLOB"
	default:
		return c.ColType
	}
}

func (p *SQLite) buildForeignKey(t fizz.Table, fk fizz.ForeignKey, onCreate bool) string {
	refs := fmt.Sprintf("%s (%s)", fk.References.Table, strings.Join(fk.References.Columns, ", "))
	s := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s", fk.Column, refs)

	if onUpdate, ok := fk.Options["on_update"]; ok {
		s += fmt.Sprintf(" ON UPDATE %s", onUpdate)
	}

	if onDelete, ok := fk.Options["on_delete"]; ok {
		s += fmt.Sprintf(" ON DELETE %s", onDelete)
	}

	if !onCreate {
		s = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s", t.Name, fk.Name, s)
	}

	return s
}
