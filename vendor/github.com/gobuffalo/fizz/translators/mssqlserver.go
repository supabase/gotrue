package translators

import (
	"fmt"
	"strings"

	"github.com/gobuffalo/fizz"
)

// MsSqlServer is a MS SqlServer-specific translator.
type MsSqlServer struct{}

// NewMsSqlServer constructs a new MsSqlServer translator.
func NewMsSqlServer() *MsSqlServer {
	return &MsSqlServer{}
}

func (MsSqlServer) Name() string {
	return "mssqlserver"
}

func (p *MsSqlServer) CreateTable(t fizz.Table) (string, error) {
	sql := []string{}
	cols := []string{}
	var s string
	for _, c := range t.Columns {
		if c.Primary {
			s = fmt.Sprintf("%s %s PRIMARY KEY", c.Name, p.colType(c))
			if c.Primary && ((c.ColType == "integer" || strings.ToLower(c.ColType) == "int") || (strings.ToLower(c.ColType) == "bigint")) {
				s = fmt.Sprintf("%s IDENTITY(1,1)", s)
			}
		} else {
			s = p.buildAddColumn(t.Name, c)
		}
		cols = append(cols, s)
	}

	primaryKeys := t.PrimaryKeys()
	if len(primaryKeys) > 1 {
		pks := make([]string, len(primaryKeys))
		for i, pk := range primaryKeys {
			pks[i] = fmt.Sprintf("[%s]", pk)
		}
		cols = append(cols, fmt.Sprintf("PRIMARY KEY(%s)", strings.Join(pks, ", ")))
	}

	s = fmt.Sprintf("CREATE TABLE %s (\n%s\n);", t.Name, strings.Join(cols, ",\n"))
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
	for _, fk := range t.ForeignKeys {
		sql = append(sql, p.buildForeignKey(t, fk))
	}
	return strings.Join(sql, "\n"), nil
}

func (p *MsSqlServer) DropTable(t fizz.Table) (string, error) {
	return fmt.Sprintf("DROP TABLE %s;", t.Name), nil
}

func (p *MsSqlServer) RenameTable(t []fizz.Table) (string, error) {
	if len(t) < 2 {
		return "", fmt.Errorf("Not enough table names supplied")
	}
	return fmt.Sprintf("EXEC sp_rename '%s', '%s';", t[0].Name, t[1].Name), nil
}

func (p *MsSqlServer) ChangeColumn(t fizz.Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", fmt.Errorf("Not enough columns supplied")
	}
	c := t.Columns[0]

	cmds := make([]string, 0)
	s := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s", t.Name, c.Name, p.colType(c))
	if c.Options["null"] == nil {
		s = fmt.Sprintf("%s NOT NULL", s)
	} else {
		s = fmt.Sprintf("%s NULL", s)
	}
	cmds = append(cmds, s)
	setDefault := c.Options["default"] != nil || c.Options["default_raw"] != nil
	if setDefault {
		dfConstraintName := fmt.Sprintf("DF_%s_%s", t.Name, c.Name)
		cmds = append(cmds, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", t.Name, dfConstraintName))
		s := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s DEFAULT", t.Name, dfConstraintName)
		if c.Options["default"] != nil {
			cmds = append(cmds, fmt.Sprintf("%s '%v' FOR %s;", s, c.Options["default"], c.Name))
		}
		if c.Options["default_raw"] != nil {
			cmds = append(cmds, fmt.Sprintf("%s %s FOR %s;", s, c.Options["default_raw"], c.Name))
		}
	}
	if len(cmds) > 0 {
		return strings.Join(cmds, "\n"), nil
	}
	return "", nil
}

func (p *MsSqlServer) AddColumn(t fizz.Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	//if _, ok := t.Columns[0].Options["first"]; ok {
	//	return "", fmt.Errorf("T-SQL does not support adding column at a specific position.")
	//}
	//
	//if _, ok := t.Columns[0].Options["after"]; ok {
	//	return "", fmt.Errorf("T-SQL does not support adding column at a specific position.")
	//}
	c := t.Columns[0]
	s := fmt.Sprintf("ALTER TABLE %s ADD %s;", t.Name, p.buildAddColumn(t.Name, c))

	return s, nil
}

func (p *MsSqlServer) DropColumn(t fizz.Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	c := t.Columns[0]
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", t.Name, c.Name), nil
}

func (p *MsSqlServer) RenameColumn(t fizz.Table) (string, error) {
	if len(t.Columns) < 2 {
		return "", fmt.Errorf("not enough columns supplied")
	}
	oc := t.Columns[0]
	nc := t.Columns[1]
	s := fmt.Sprintf("EXEC sp_rename '%s.%s', '%s', 'COLUMN';", t.Name, oc.Name, nc.Name)
	return s, nil
}

func (p *MsSqlServer) AddIndex(t fizz.Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", fmt.Errorf("not enough indexes supplied")
	}
	i := t.Indexes[0]
	s := fmt.Sprintf("CREATE INDEX %s ON %s (%s);", i.Name, t.Name, strings.Join(i.Columns, ", "))
	if i.Unique {
		s = strings.Replace(s, "CREATE", "CREATE UNIQUE", 1)
	}
	return s, nil
}

func (p *MsSqlServer) DropIndex(t fizz.Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", fmt.Errorf("not enough indexes supplied")
	}
	i := t.Indexes[0]
	return fmt.Sprintf("DROP INDEX %s ON %s;", i.Name, t.Name), nil
}

func (p *MsSqlServer) RenameIndex(t fizz.Table) (string, error) {
	ix := t.Indexes
	if len(ix) < 2 {
		return "", fmt.Errorf("not enough indexes supplied")
	}
	oi := ix[0]
	ni := ix[1]
	return fmt.Sprintf("EXEC sp_rename '%s.%s', '%s', 'INDEX';", t.Name, oi.Name, ni.Name), nil
}

func (p *MsSqlServer) AddForeignKey(t fizz.Table) (string, error) {
	if len(t.ForeignKeys) == 0 {
		return "", fmt.Errorf("not enough foreign keys supplied")
	}

	return p.buildForeignKey(t, t.ForeignKeys[0]), nil
}

func (p *MsSqlServer) DropForeignKey(t fizz.Table) (string, error) {
	if len(t.ForeignKeys) == 0 {
		return "", fmt.Errorf("not enough foreign keys supplied")
	}

	fk := t.ForeignKeys[0]

	var ifExists string
	if v, ok := fk.Options["if_exists"]; ok && v.(bool) {
		ifExists = "IF EXISTS"
	}

	s := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s %s;", t.Name, ifExists, fk.Name)
	return s, nil
}

func (p *MsSqlServer) buildAddColumn(tableName string, c fizz.Column) string {
	s := fmt.Sprintf("%s %s", c.Name, p.colType(c))
	if c.Options["null"] == nil {
		s = fmt.Sprintf("%s NOT NULL", s)
	}
	setDefault := c.Options["default"] != nil || c.Options["default_raw"] != nil
	if setDefault {
		dfConstraintName := fmt.Sprintf("DF_%s_%s", tableName, c.Name)
		if c.Options["default"] != nil {
			s = fmt.Sprintf("%s CONSTRAINT %s DEFAULT '%v'", s, dfConstraintName, c.Options["default"])
		}
		if c.Options["default_raw"] != nil {
			s = fmt.Sprintf("%s CONSTRAINT %s DEFAULT %s", s, dfConstraintName, c.Options["default_raw"])
		}
	}

	return s
}

func (p *MsSqlServer) colType(c fizz.Column) string {
	switch c.ColType {
	case "integer":
		return "INT"
	case "string":
		s := "255"
		if c.Options["size"] != nil {
			s = fmt.Sprintf("%d", c.Options["size"])
		}
		return fmt.Sprintf("NVARCHAR (%s)", s)
	case "uuid":
		return "uniqueidentifier"
	case "blob":
		return "VARBINARY(MAX)"
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
	case "timestamp":
		return "DATETIME"
	case "boolean":
		return "BIT"
	default:
		return c.ColType
	}
}

func (p *MsSqlServer) buildForeignKey(t fizz.Table, fk fizz.ForeignKey) string {
	refs := fmt.Sprintf("%s (%s)", fk.References.Table, strings.Join(fk.References.Columns, ", "))
	s := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s", fk.Column, refs)
	s = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;", t.Name, fk.Name, s)
	return s
}
