package translators

import (
	"database/sql"
	"fmt"

	"github.com/gobuffalo/fizz"
)

type cockroachForeignKeyListInfo struct {
	Name      string `db:"name"`
	Column    string `db:"column_name"`
	TableRef  string `db:"referenced_table_name"`
	ColumnRef string `db:"referenced_column_name"`
	OnUpdate  string `db:"on_update"`
	OnDelete  string `db:"on_delete"`
	Match     string `db:"match"`
}

type cockroachIndexListInfo struct {
	Name      string `db:"name"`
	NonUnique bool   `db:"non_unique"`
}

type cockroachIndexInfo struct {
	Name      string `db:"name"`
	Direction string `db:"direction"`
}

type cockroachTableInfo struct {
	Name    string      `db:"column_name"`
	Type    string      `db:"data_type"`
	NotNull bool        `db:"not_null"`
	Default interface{} `db:"column_default"`
	PK      bool        `db:"pk"`
}

func (t cockroachTableInfo) ToColumn() fizz.Column {
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
		c.Options["default_raw"] = fmt.Sprint(t.Default) // strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%s", t.Default), "'"), "'")
	}
	return c
}

type cockroachSchema struct {
	Schema
}

func (p *cockroachSchema) Build() error {
	var err error
	db, err := sql.Open("pgx", p.URL)
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := db.Query("SELECT table_name as name FROM information_schema.tables;")
	if err != nil {
		return err
	}
	for res.Next() {
		table := &fizz.Table{
			Columns: []fizz.Column{},
			Indexes: []fizz.Index{},
		}
		err = res.Scan(&table.Name)
		if err != nil {
			return err
		}
		if table.Name != "cockroach_sequence" {
			err = p.buildTableData(table, db)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (p *cockroachSchema) buildTableData(table *fizz.Table, db *sql.DB) error {
	prag := fmt.Sprintf(`SELECT c.column_name, 
	c.data_type, 
	(c.is_nullable = 'NO') as "not_null",
	c.column_default,
	(tc.table_schema IS NOT NULL)::bool AS "pk"
	FROM information_schema.columns AS c
	LEFT JOIN information_schema.key_column_usage as kcu
		ON ((c.table_schema = kcu.table_schema)
		AND (c.table_name = kcu.table_name)
		AND (c.column_name = kcu.column_name))
	LEFT JOIN information_schema.table_constraints AS tc
		ON ((tc.table_schema = kcu.table_schema)
		AND (tc.table_name = kcu.table_name)
		AND (tc.constraint_name = kcu.constraint_name))
		AND (tc.constraint_name = 'primary')
	WHERE c.table_name = '%s';`, table.Name)

	res, err := db.Query(prag)
	if err != nil {
		return err
	}
	defer res.Close()

	for res.Next() {
		ti := cockroachTableInfo{}
		err = res.Scan(&ti.Name, &ti.Type, &ti.NotNull, &ti.Default, &ti.PK)
		if err != nil {
			return err
		}
		table.Columns = append(table.Columns, ti.ToColumn())
	}
	err = p.buildTableIndexes(table, db)
	if err != nil {
		return err
	}
	err = p.buildForeignKeyIndexes(table, db)
	if err != nil {
		return err
	}
	p.schema[table.Name] = table
	return nil
}

func (p *cockroachSchema) buildTableIndexes(t *fizz.Table, db *sql.DB) error {
	prag := fmt.Sprintf(`
SELECT
    DISTINCT index_name AS name,
                     (non_unique = 'YES') AS non_unique
FROM
     information_schema.statistics
WHERE
      table_name = '%s';
`, t.Name)
	res, err := db.Query(prag)
	if err != nil {
		return err
	}
	defer res.Close()

	for res.Next() {
		li := cockroachIndexListInfo{}
		err = res.Scan(&li.Name, &li.NonUnique)
		if err != nil {
			return err
		}

		i := fizz.Index{
			Name:    li.Name,
			Unique:  !li.NonUnique,
			Columns: []string{},
		}

		prag = fmt.Sprintf("SELECT column_name as name, direction FROM information_schema.statistics where index_name = '%s' and implicit = 'NO';", i.Name)
		iires, err := db.Query(prag)
		if err != nil {
			return err
		}

		for iires.Next() {
			ii := cockroachIndexInfo{}
			err = iires.Scan(&ii.Name, &ii.Direction)
			if err != nil {
				return err
			}
			i.Columns = append(i.Columns, ii.Name)
		}

		t.Indexes = append(t.Indexes, i)
	}
	return nil
}

func (p *cockroachSchema) buildForeignKeyIndexes(t *fizz.Table, db *sql.DB) error {
	prag := fmt.Sprintf(`
SELECT
	fk.constraint_name,
	fk.referenced_table_name,
	col.column_name,
	fk.update_rule,
	fk.delete_rule,
	fk.match_option
FROM
	information_schema.referential_constraints as fk
INNER JOIN
	information_schema.key_column_usage as col
ON
	col.constraint_name = fk.constraint_name
WHERE
	fk.table_name = '%s'
;`, t.Name)
	res, err := db.Query(prag)
	if err != nil {
		return err
	}
	defer res.Close()

	for res.Next() {
		li := cockroachForeignKeyListInfo{}
		err = res.Scan(
			&li.Name,
			&li.TableRef,
			&li.Column,
			&li.OnUpdate,
			&li.OnDelete,
			&li.Match)
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

		if li.Match != "" {
			options["match"] = li.Match
		}

		ref := fizz.ForeignKeyRef{
			Table:   li.TableRef,
			Columns: []string{},
		}

		prag = fmt.Sprintf(`
SELECT
	column_name as referenced_column_name
FROM
	information_schema.constraint_column_usage as ref
WHERE 
	constraint_name = '%s'
;`, li.Name)
		iires, err := db.Query(prag)
		if err != nil {
			return err
		}

		for iires.Next() {
			ii := cockroachForeignKeyListInfo{}
			err = iires.Scan(&ii.ColumnRef)
			if err != nil {
				return err
			}
			ref.Columns = append(ref.Columns, ii.ColumnRef)
		}

		i := fizz.ForeignKey{
			Name:       li.Name,
			Column:     li.Column,
			References: ref,
			Options:    options,
		}

		t.ForeignKeys = append(t.ForeignKeys, i)
	}
	return nil
}
