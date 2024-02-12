package translators

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/gobuffalo/fizz"
)

type sqliteIndexListInfo struct {
	Seq     int    `db:"seq"`
	Name    string `db:"name"`
	Unique  bool   `db:"unique"`
	Origin  string `db:"origin"`
	Partial string `db:"partial"`
}

type sqliteForeignKeyListInfo struct {
	ID       int    `db:"id"`
	Seq      int    `db:"seq"`
	From     string `db:"from"`
	To       string `db:"to"`
	Table    string `db:"table"`
	OnUpdate string `db:"on_update"`
	OnDelete string `db:"on_delete"`
	Match    string `db:"match"`
}

type sqliteIndexInfo struct {
	Seq  int    `db:"seqno"`
	CID  int    `db:"cid"`
	Name string `db:"name"`
}

type sqliteTableInfo struct {
	CID     int         `db:"cid"`
	Name    string      `db:"name"`
	Type    string      `db:"type"`
	NotNull bool        `db:"notnull"`
	Default interface{} `db:"dflt_value"`
	PK      bool        `db:"pk"`
}

func (t sqliteTableInfo) ToColumn() fizz.Column {
	c := fizz.Column{
		Name:    t.Name,
		ColType: t.Type,
		Primary: t.PK,
		Options: fizz.Options{},
	}
	if !t.NotNull {
		c.Options["null"] = true
	}
	if t.Default != nil {
		c.Options["default"] = strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%s", t.Default), "'"), "'")
	}
	return c
}

type sqliteSchema struct {
	Schema
}

func (p *sqliteSchema) Build() error {
	var err error
	db, err := sql.Open("sqlite3", p.URL)
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := db.Query("SELECT name FROM sqlite_master WHERE type='table';")
	if err != nil {
		return err
	}
	defer res.Close()

	for res.Next() {
		table := &fizz.Table{
			Columns: []fizz.Column{},
			Indexes: []fizz.Index{},
		}
		err = res.Scan(&table.Name)
		if err != nil {
			return err
		}
		if table.Name != "sqlite_sequence" {
			err = p.buildTableData(table, db)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (p *sqliteSchema) buildTableData(table *fizz.Table, db *sql.DB) error {
	prag := fmt.Sprintf(`SELECT "cid", "name", "type", "notnull", "dflt_value", "pk" FROM pragma_table_info('%s')`, table.Name)

	res, err := db.Query(prag)
	if err != nil {
		return err
	}
	defer res.Close()

	for res.Next() {
		ti := sqliteTableInfo{}
		err = res.Scan(&ti.CID, &ti.Name, &ti.Type, &ti.NotNull, &ti.Default, &ti.PK)
		if err != nil {
			return err
		}
		table.Columns = append(table.Columns, ti.ToColumn())
	}
	err = p.buildTableIndexes(table, db)
	if err != nil {
		return err
	}
	err = p.buildTableForeignKeyIndexes(table, db)
	if err != nil {
		return err
	}
	p.schema[table.Name] = table
	return nil
}

func (p *sqliteSchema) buildTableIndexes(t *fizz.Table, db *sql.DB) error {
	// This ignores all internal SQLite keys which are prefixed with `sqlite_` as explained here:
	// https://www.sqlite.org/fileformat2.html#intschema
	prag := fmt.Sprintf(`SELECT "seq", "name", "unique", "origin", "partial" FROM pragma_index_list('%s') WHERE "name" NOT LIKE 'sqlite_%%'`, t.Name)
	res, err := db.Query(prag)
	if err != nil {
		return err
	}
	defer res.Close()

	for res.Next() {
		li := sqliteIndexListInfo{}
		err = res.Scan(&li.Seq, &li.Name, &li.Unique, &li.Origin, &li.Partial)
		if err != nil {
			return err
		}

		i := fizz.Index{
			Name:    li.Name,
			Unique:  li.Unique,
			Columns: []string{},
		}

		prag = fmt.Sprintf(`SELECT "seqno", "cid", "name" FROM pragma_index_info('%s');`, i.Name)
		iires, err := db.Query(prag)
		if err != nil {
			return err
		}
		defer iires.Close()

		for iires.Next() {
			ii := sqliteIndexInfo{}
			err = iires.Scan(&ii.Seq, &ii.CID, &ii.Name)
			if err != nil {
				return err
			}
			i.Columns = append(i.Columns, ii.Name)
		}

		t.Indexes = append(t.Indexes, i)

	}
	return nil
}

var tmpTable = regexp.MustCompile("^_(.*)_tmp$")

func canonicalizeSQLiteTable(table string) string {
	matches := tmpTable.FindAllStringSubmatch(table, 1)
	if len(matches) == 1 && len(matches[0]) == 2 {
		return matches[0][1]
	}
	return table
}

func (p *sqliteSchema) buildTableForeignKeyIndexes(t *fizz.Table, db *sql.DB) error {
	// This ignores all internal SQLite keys which are prefixed with `sqlite_` as explained here:
	// https://www.sqlite.org/fileformat2.html#intschema
	prag := fmt.Sprintf(`SELECT "seq", "table", "from", "to", "on_update", "on_delete", "match" FROM pragma_foreign_key_list('%s')`, t.Name)
	res, err := db.Query(prag)
	if err != nil {
		return err
	}
	defer res.Close()

	foreignKeys := []fizz.ForeignKey{}
	for res.Next() {
		li := sqliteForeignKeyListInfo{}
		err = res.Scan(&li.Seq, &li.Table, &li.From, &li.To, &li.OnUpdate, &li.OnDelete, &li.Match)
		if err != nil {
			return err
		}

		options := map[string]interface{}{}
		if li.OnDelete != "" {
			options["on_delete"] = li.OnDelete
		}

		if li.OnUpdate != "" {
			options["on_update"] = li.OnUpdate
		}

		i := fizz.ForeignKey{
			Column: li.From,
			References: fizz.ForeignKeyRef{
				Table:   canonicalizeSQLiteTable(li.Table),
				Columns: []string{li.To},
			},
			Options: options,
		}
		i.Name = fmt.Sprintf("%s_%s_%s_fk", t.Name, i.References.Table, strings.Join(i.References.Columns, "_"))

		foreignKeys = append(foreignKeys, i)
	}
	t.ForeignKeys = foreignKeys
	return nil
}
