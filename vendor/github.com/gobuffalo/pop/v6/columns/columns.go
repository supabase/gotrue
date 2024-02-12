package columns

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Columns represent a list of columns, related to a given table.
type Columns struct {
	Cols       map[string]*Column
	lock       *sync.RWMutex
	TableName  string
	TableAlias string
	IDField    string
}

// Add a column to the list.
func (c *Columns) Add(names ...string) []*Column {
	var ret []*Column
	c.lock.Lock()

	tableAlias := c.TableAlias
	if tableAlias == "" {
		tableAlias = c.TableName
	}

	for _, name := range names {

		var xs []string
		var col *Column
		ss := ""
		//support for distinct xx, or distinct on (field) table.fields
		if strings.HasSuffix(name, ",r") || strings.HasSuffix(name, ",w") {
			xs = []string{name[0 : len(name)-2], name[len(name)-1:]}
		} else {
			xs = []string{name}
		}

		xs[0] = strings.TrimSpace(xs[0])
		//eg: id id2 - select id as id2
		// also distinct columnname
		// and distinct on (column1) column2
		if strings.Contains(strings.ToUpper(xs[0]), " AS ") {
			//eg: select id as id2
			i := strings.LastIndex(strings.ToUpper(xs[0]), " AS ")
			ss = xs[0]
			xs[0] = xs[0][i+4 : len(xs[0])] //get id2
		} else if strings.Contains(xs[0], " ") {
			i := strings.LastIndex(name, " ")
			ss = xs[0]
			xs[0] = xs[0][i+1 : len(xs[0])] //get id2
		}

		col = c.Cols[xs[0]]
		if col == nil {
			if ss == "" {
				ss = xs[0]
				if tableAlias != "" {
					ss = fmt.Sprintf("%s.%s", tableAlias, ss)
				}
			}

			col = &Column{
				Name:      xs[0],
				SelectSQL: ss,
				Readable:  true,
				Writeable: true,
			}

			if len(xs) > 1 {
				if xs[1] == "r" {
					col.Writeable = false
				} else if xs[1] == "w" {
					col.Readable = false
				}
			} else if col.Name == c.IDField {
				col.Writeable = false
			}

			c.Cols[col.Name] = col
		}
		ret = append(ret, col)
	}

	c.lock.Unlock()
	return ret
}

// Remove a column from the list.
func (c *Columns) Remove(names ...string) {
	for _, name := range names {
		xs := strings.Split(name, ",")
		name = xs[0]
		delete(c.Cols, name)
	}
}

// Writeable gets a list of the writeable columns from the column list.
func (c Columns) Writeable() *WriteableColumns {
	w := &WriteableColumns{NewColumnsWithAlias(c.TableName, c.TableAlias, c.IDField)}
	for _, col := range c.Cols {
		if col.Writeable {
			w.Cols[col.Name] = col
		}
	}
	return w
}

// Readable gets a list of the readable columns from the column list.
func (c Columns) Readable() *ReadableColumns {
	w := &ReadableColumns{NewColumnsWithAlias(c.TableName, c.TableAlias, c.IDField)}
	for _, col := range c.Cols {
		if col.Readable {
			w.Cols[col.Name] = col
		}
	}
	return w
}

// colNames returns a slice of column names in a deterministic order
func (c Columns) colNames() []string {
	var xs []string
	for _, t := range c.Cols {
		xs = append(xs, t.Name)
	}
	sort.Strings(xs)
	return xs
}

type quoter interface {
	Quote(key string) string
}

// QuotedString gives the columns list quoted with the given quoter function.
func (c Columns) QuotedString(quoter quoter) string {
	xs := c.colNames()
	for i, n := range xs {
		xs[i] = quoter.Quote(n)
	}
	return strings.Join(xs, ", ")
}

func (c Columns) String() string {
	xs := c.colNames()
	return strings.Join(xs, ", ")
}

// SymbolizedString returns a list of tokens (:token) to bind
// a value to an INSERT query.
func (c Columns) SymbolizedString() string {
	xs := c.colNames()
	for i, n := range xs {
		xs[i] = ":" + n
	}
	return strings.Join(xs, ", ")
}

// NewColumns constructs a list of columns for a given table name.
func NewColumns(tableName, idField string) Columns {
	return NewColumnsWithAlias(tableName, "", idField)
}

// NewColumnsWithAlias constructs a list of columns for a given table
// name, using a given alias for the table.
func NewColumnsWithAlias(tableName, tableAlias, idField string) Columns {
	return Columns{
		lock:       &sync.RWMutex{},
		Cols:       map[string]*Column{},
		TableName:  tableName,
		TableAlias: tableAlias,
		IDField:    idField,
	}
}
