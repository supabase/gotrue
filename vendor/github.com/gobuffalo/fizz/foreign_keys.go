package fizz

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ForeignKeyRef struct {
	Table   string
	Columns []string
}

type ForeignKey struct {
	Name       string
	Column     string
	References ForeignKeyRef
	Options    Options
}

func (f ForeignKey) String() string {
	refs := fmt.Sprintf(`{"%s": ["%s"]}`, f.References.Table, strings.Join(f.References.Columns, `", "`))
	var opts map[string]interface{}
	if f.Options == nil {
		opts = make(map[string]interface{})
	} else {
		opts = f.Options
	}

	o := make([]string, 0, len(opts))
	for k, v := range opts {
		vv, _ := json.Marshal(v)
		o = append(o, fmt.Sprintf("%s: %s", k, string(vv)))
	}
	sort.SliceStable(o, func(i, j int) bool { return o[i] < o[j] })
	return fmt.Sprintf(`t.ForeignKey("%s", %s, {%s})`, f.Column, refs, strings.Join(o, ", "))
}

func (f fizzer) AddForeignKey(table string, column string, refs interface{}, options Options) error {
	fkr, err := parseForeignKeyRef(refs)
	if err != nil {
		return err
	}
	fk := ForeignKey{
		Column:     column,
		References: fkr,
		Options:    options,
	}

	if options["name"] != nil {
		var ok bool
		fk.Name, ok = options["name"].(string)
		if !ok {
			return fmt.Errorf(`expected options field "name" to be of type "string" but got "%T"`, options["name"])
		}
	} else {
		fk.Name = fmt.Sprintf("%s_%s_%s_fk", table, fk.References.Table, strings.Join(fk.References.Columns, "_"))
	}

	return f.add(f.Bubbler.AddForeignKey(Table{
		Name:        table,
		ForeignKeys: []ForeignKey{fk},
	}))
}

func (f fizzer) DropForeignKey(table string, fk string, options Options) error {
	return f.add(f.Bubbler.DropForeignKey(Table{
		Name: table,
		ForeignKeys: []ForeignKey{
			{
				Name:    fk,
				Options: options,
			},
		},
	}))
}

func parseForeignKeyRef(refs interface{}) (ForeignKeyRef, error) {
	fkr := ForeignKeyRef{}
	refMap, ok := refs.(map[string]interface{})
	if !ok {
		return fkr, fmt.Errorf(`invalid references format %s\nmust be "{"table": ["colum1", "column2"]}"`, refs)
	}
	if len(refMap) != 1 {
		return fkr, fmt.Errorf("only one table is supported as Foreign key reference")
	}
	for table, columns := range refMap {
		fkr.Table = table
		for _, c := range columns.([]interface{}) {
			fkr.Columns = append(fkr.Columns, fmt.Sprintf("%s", c))
		}
	}

	return fkr, nil
}
