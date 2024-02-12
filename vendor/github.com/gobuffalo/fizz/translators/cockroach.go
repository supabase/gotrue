package translators

import (
	"fmt"
	"strings"

	"github.com/gobuffalo/fizz"
)

type Cockroach struct {
	Schema SchemaQuery
}

func NewCockroach(url string, name string) *Cockroach {
	schema := &cockroachSchema{
		Schema{
			URL:    url,
			Name:   name,
			schema: map[string]*fizz.Table{},
		},
	}
	schema.Builder = schema
	return &Cockroach{Schema: schema}
}

func (Cockroach) Name() string {
	return "cockroach"
}

func (p *Cockroach) CreateTable(t fizz.Table) (string, error) {
	p.Schema.SetTable(&t)
	sql := []string{}
	cols := []string{}
	var s string
	for _, c := range t.Columns {
		if c.Primary {
			switch c.ColType {
			case "string", "uuid":
			case "integer", "INT", "int":
				c.ColType = "SERIAL"
			default:
				return "", fmt.Errorf("can not use %s as a primary key", c.ColType)
			}
		}
		cols = append(cols, p.buildAddColumn(c))
		if c.Primary {
			cols = append(cols, fmt.Sprintf(`PRIMARY KEY("%s")`, c.Name))
		}
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

	s = fmt.Sprintf("CREATE TABLE \"%s\" (\n%s\n);COMMIT TRANSACTION;BEGIN TRANSACTION;", t.Name, strings.Join(cols, ",\n"))
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

func (p *Cockroach) DropTable(t fizz.Table) (string, error) {
	p.Schema.Delete(t.Name)
	return fmt.Sprintf("DROP TABLE \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", t.Name), nil
}

func (p *Cockroach) RenameTable(t []fizz.Table) (string, error) {
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

	return fmt.Sprintf("ALTER TABLE \"%s\" RENAME TO \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", oldName, newName), nil
}

func (p *Cockroach) ChangeColumn(t fizz.Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	c := t.Columns[0]

	tableInfo, err := p.Schema.TableInfo(t.Name)

	if err != nil {
		return "", err
	}

	for i := range tableInfo.Columns {
		if tableInfo.Columns[i].Name == t.Columns[0].Name {
			tableInfo.Columns[i] = c
			break
		}
	}

	sql := []string{}
	s, err := p.withTempColumn(t.Name, c.Name, func(table fizz.Table, origCol fizz.Column, tempCol string) (string, error) {
		var notNullWorkaround bool
		if c.Options["default_raw"] == nil && c.Options["default"] == nil && c.Options["null"] == nil {
			// This happens when the original column has "NOT NULL" and the changed column also has "NOT NULL" and no "DEFAULT".
			// In those cases, if the table has data already, CockroachDB will fail with an error mandating that a column can not
			// be added to the table unless it is either "NULL" or has a "DEFAULT". By first using NULL and then setting the
			// "NOT NULL" constraint, this problem is worked around.
			notNullWorkaround = true
		}

		newCol := p.buildChangeColumn(origCol, c)
		err1 := p.Schema.ReplaceColumn(table.Name, origCol.Name, newCol)
		if err1 != nil {
			return "", err1
		}

		if notNullWorkaround {
			newCol.Options["null"] = true
		}

		createColumnSQL := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN %s;COMMIT TRANSACTION;BEGIN TRANSACTION;`, table.Name, p.buildAddColumn(newCol))
		ins := fmt.Sprintf(`UPDATE "%s" SET "%s" = "%s";COMMIT TRANSACTION;BEGIN TRANSACTION;`, t.Name, c.Name, tempCol)

		sql := []string{createColumnSQL, ins}
		if notNullWorkaround {
			newCol.Options["null"] = nil
			sql = append(sql, fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET NOT NULL;COMMIT TRANSACTION;BEGIN TRANSACTION;`, t.Name, newCol.Name))
		}

		return strings.Join(sql, "\n"), nil
	})

	if err != nil {
		return "", err
	}

	sql = append(sql, s)

	return strings.Join(sql, "\n"), nil
}

func (p *Cockroach) AddColumn(t fizz.Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	c := t.Columns[0]
	s := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN %s;COMMIT TRANSACTION;BEGIN TRANSACTION;", t.Name, p.buildAddColumn(c))

	// Update schema cache if we can
	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err == nil {
		found := false
		for i := range tableInfo.Columns {
			if tableInfo.Columns[i].Name == c.Name {
				tableInfo.Columns[i] = c
				found = true
				break
			}
		}
		if !found {
			tableInfo.Columns = append(tableInfo.Columns, c)
		}
	}

	return s, nil
}

func (p *Cockroach) DropColumn(t fizz.Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	c := t.Columns[0]
	p.Schema.DeleteColumn(t.Name, c.Name)
	return fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", t.Name, c.Name), nil
}

func (p *Cockroach) RenameColumn(t fizz.Table) (string, error) {
	if len(t.Columns) < 2 {
		return "", fmt.Errorf("not enough columns supplied")
	}

	oc := t.Columns[0]
	nc := t.Columns[1]

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}

	for ic, c := range tableInfo.Columns {
		if c.Name == oc.Name {
			tableInfo.Columns[ic].Name = nc.Name
		}
	}

	s := fmt.Sprintf("ALTER TABLE \"%s\" RENAME COLUMN \"%s\" TO \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", t.Name, oc.Name, nc.Name)
	return s, nil
}

func (p *Cockroach) AddIndex(t fizz.Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", fmt.Errorf("Not enough indexes supplied")
	}
	i := t.Indexes[0]
	s := fmt.Sprintf("CREATE INDEX \"%s\" ON \"%s\" (%s);COMMIT TRANSACTION;BEGIN TRANSACTION;", i.Name, t.Name, strings.Join(i.Columns, ", "))
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

func (p *Cockroach) DropIndex(t fizz.Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", fmt.Errorf("not enough indexes supplied")
	}
	i := t.Indexes[0]

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

	return fmt.Sprintf("DROP INDEX IF EXISTS \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", i.Name), nil
}

func (p *Cockroach) RenameIndex(t fizz.Table) (string, error) {
	ix := t.Indexes
	if len(ix) < 2 {
		return "", fmt.Errorf("not enough indexes supplied")
	}
	oi := ix[0]
	ni := ix[1]

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}

	for _, c := range tableInfo.Indexes {
		if c.Name == oi.Name {
			c.Name = ni.Name
		}
	}

	return fmt.Sprintf("ALTER INDEX \"%s\"@\"%s\" RENAME TO \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", t.Name, oi.Name, ni.Name), nil
}

func (p *Cockroach) AddForeignKey(t fizz.Table) (string, error) {
	if len(t.ForeignKeys) == 0 {
		return "", fmt.Errorf("not enough foreign keys supplied")
	}

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}
	tableInfo.ForeignKeys = append(tableInfo.ForeignKeys, t.ForeignKeys[0])

	return p.buildForeignKey(t, t.ForeignKeys[0], false), nil
}

func (p *Cockroach) DropForeignKey(t fizz.Table) (string, error) {
	if len(t.ForeignKeys) == 0 {
		return "", fmt.Errorf("not enough foreign keys supplied")
	}

	fk := t.ForeignKeys[0]

	tableInfo, err := p.Schema.TableInfo(t.Name)
	if err != nil {
		return "", err
	}
	newFKs := []fizz.ForeignKey{}
	for _, key := range tableInfo.ForeignKeys {
		if key.Name != fk.Name {
			newFKs = append(newFKs, key)
		}
	}
	tableInfo.ForeignKeys = newFKs

	var ifExists string
	if v, ok := fk.Options["if_exists"]; ok && v.(bool) {
		ifExists = "IF EXISTS "
	}

	s := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s\"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", p.escapeIdentifier(t.Name), ifExists, fk.Name)
	return s, nil
}

func (p *Cockroach) buildAddColumn(c fizz.Column) string {
	s := fmt.Sprintf("\"%s\" %s", c.Name, p.colType(c))

	if c.Options["null"] == nil || c.Primary {
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

func (p *Cockroach) buildChangeColumn(oldCol fizz.Column, c fizz.Column) fizz.Column {
	newCol := fizz.Column{
		Name:    c.Name,
		ColType: oldCol.ColType,
		Options: oldCol.Options,
		Primary: oldCol.Primary,
	}

	if c.Options["default"] != nil {
		newCol.Options["default"] = c.Options["default"]
	}
	if c.Options["null"] != nil {
		newCol.Options["null"] = c.Options["null"]
	}
	if c.Options["default_raw"] != nil {
		newCol.Options["default_raw"] = c.Options["default_raw"]
	}

	return newCol
}

func (p *Cockroach) withTempTable(table string, fn func(fizz.Table) (string, error)) (string, error) {
	tempTable := fizz.Table{Name: fmt.Sprintf("_%s_tmp", table)}

	sql := []string{fmt.Sprintf("ALTER TABLE \"%s\" RENAME TO \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", table, tempTable.Name)}
	s, err := fn(tempTable)
	if err != nil {
		return "", err
	}
	sql = append(sql, s, fmt.Sprintf("DROP TABLE \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", tempTable.Name))

	return strings.Join(sql, "\n"), nil
}

func (p *Cockroach) withTempColumn(tableName string, column string, fn func(fizz.Table, fizz.Column, string) (string, error)) (string, error) {
	table, err := p.Schema.TableInfo(tableName)
	if err != nil {
		return "", err
	}
	col, err1 := p.Schema.ColumnInfo(tableName, column)
	if err1 != nil {
		return "", err1
	}

	var sql []string
	var recreateForeignKeys []fizz.ForeignKey
	for _, i := range table.ForeignKeys {
		if i.Column == column {
			s, err := p.DropForeignKey(fizz.Table{
				Name:        table.Name,
				ForeignKeys: []fizz.ForeignKey{i},
			})
			if err != nil {
				return "", err
			}
			sql = append(sql, s)
			recreateForeignKeys = append(recreateForeignKeys, i)
		}
	}

	var recreateIndexes []fizz.Index
	for _, i := range table.Indexes {
		var found bool

		for _, ic := range i.Columns {
			if ic == column {
				found = true
				break
			}
		}

		if found {
			s, err := p.DropIndex(fizz.Table{
				Name:    table.Name,
				Indexes: []fizz.Index{i},
			})
			if err != nil {
				return "", err
			}
			sql = append(sql, s)
			recreateIndexes = append(recreateIndexes, i)
		}
	}

	tempCol := fmt.Sprintf("_%s_tmp", column)

	sql = append(sql, fmt.Sprintf("ALTER TABLE \"%s\" RENAME COLUMN \"%s\" TO \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", tableName, column, tempCol))

	s, err := fn(*table, *col, tempCol)
	if err != nil {
		return "", err
	}
	sql = append(sql, s, fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN \"%s\";COMMIT TRANSACTION;BEGIN TRANSACTION;", tableName, tempCol))

	for _, i := range recreateIndexes {
		s, err := p.AddIndex(fizz.Table{
			Name:    table.Name,
			Indexes: []fizz.Index{i},
		})
		if err != nil {
			return "", err
		}
		sql = append(sql, s)
	}

	for _, i := range recreateForeignKeys {
		s, err := p.AddForeignKey(fizz.Table{
			Name:        table.Name,
			ForeignKeys: []fizz.ForeignKey{i},
		})
		if err != nil {
			return "", err
		}
		sql = append(sql, s)
	}

	return strings.Join(sql, "\n"), nil
}

func (p *Cockroach) colType(c fizz.Column) string {
	switch c.ColType {
	case "string":
		s := "255"
		if c.Options["size"] != nil {
			s = fmt.Sprintf("%d", c.Options["size"])
		}
		return fmt.Sprintf("VARCHAR (%s)", s)
	case "uuid":
		return "UUID"
	case "time", "datetime":
		return "timestamp"
	case "blob":
		return "BYTES"
	case "float", "decimal":
		if c.Options["precision"] != nil {
			precision := c.Options["precision"]
			if c.Options["scale"] != nil {
				scale := c.Options["scale"]
				return fmt.Sprintf("DECIMAL(%d,%d)", precision, scale)
			}
			return fmt.Sprintf("DECIMAL(%d)", precision)
		}

		return "DECIMAL"
	default:
		return c.ColType
	}
}

func (p *Cockroach) buildForeignKey(t fizz.Table, fk fizz.ForeignKey, onCreate bool) string {
	rcols := []string{}
	for _, c := range fk.References.Columns {
		rcols = append(rcols, fmt.Sprintf("\"%s\"", c))
	}
	refs := fmt.Sprintf("%s (%s)", p.escapeIdentifier(fk.References.Table), strings.Join(rcols, ", "))
	s := fmt.Sprintf("CONSTRAINT \"%s\" FOREIGN KEY (\"%s\") REFERENCES %s", fk.Name, fk.Column, refs)

	if onUpdate, ok := fk.Options["on_update"]; ok {
		s += fmt.Sprintf(" ON UPDATE %s", onUpdate)
	}

	if onDelete, ok := fk.Options["on_delete"]; ok {
		s += fmt.Sprintf(" ON DELETE %s", onDelete)
	}

	if !onCreate {
		s = fmt.Sprintf("ALTER TABLE %s ADD %s;COMMIT TRANSACTION;BEGIN TRANSACTION;", p.escapeIdentifier(t.Name), s)
	}

	return s
}

func (Cockroach) escapeIdentifier(s string) string {
	if !strings.ContainsRune(s, '.') {
		return fmt.Sprintf("\"%s\"", s)
	}
	parts := strings.Split(s, ".")
	for _, p := range parts {
		p = fmt.Sprintf("\"%s\"", p)
	}
	return strings.Join(parts, ".")
}
