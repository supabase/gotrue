# Fizz

[![Actions Status](https://github.com/gobuffalo/fizz/workflows/Tests/badge.svg)](https://github.com/gobuffalo/fizz/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gobuffalo/fizz.svg)](https://pkg.go.dev/github.com/gobuffalo/fizz)

A Common DSL for Migrating Databases


## Supported Database Engines

Fizz supports minimum supported version of all supported database engines.
Currently, the following database engines are officially supported. (Since
Fizz is used with the migration feature of Pop, supported databases and the
versions are correlated with Pop.)

* PostgreSQL 10
* MySQL 5.7 / MariaDB 10.3
* SQLite3 3.22
* CockroachDB v21.1
* MSSQL 2017 (not fully supported)


## Usage

### Create a Table

``` javascript
create_table("users") {
  t.Column("id", "integer", {primary: true})
  t.Column("email", "string", {})
  t.Column("twitter_handle", "string", {"size": 50})
  t.Column("age", "integer", {"default": 0})
  t.Column("admin", "bool", {"default": false})
  t.Column("company_id", "uuid", {"default_raw": "uuid_generate_v1()"})
  t.Column("bio", "text", {"null": true})
  t.Column("joined_at", "timestamp", {})
  t.Index("email", {"unique": true})
}

create_table("todos") {
  t.Column("user_id", "integer", {})
  t.Column("title", "string", {"size": 100})
  t.Column("details", "text", {"null": true})
  t.ForeignKey("user_id", {"users": ["id"]}, {"on_delete": "cascade"})
}
```

The `id` column don't have to be an integer. For instance, your can use an UUID type instead:

```javascript
create_table("users") {
  t.Column("id", "uuid", {primary: true})
  // ...
}
```

By default, fizz will generate two `timestamp` columns: `created_at` and `updated_at`.

The `t.Columns` method takes the following arguments: name of the column, the type of the field, and finally the last argument is any options you want to set on that column.

#### <a name="column-info"></a> "Common" Types:

* `string`
* `text`
* `timestamp`, `time`, `datetime`
* `integer`
* `bool`
* `uuid`

Any other type passed it will be be passed straight through to the underlying database.

For example for PostgreSQL you could pass `jsonb`and it will be supported, however, SQLite will yell very loudly at you if you do the same thing!

#### Supported Options:

* `size` - The size of the column. For example if you wanted a `varchar(50)` in Postgres you would do: `t.Column("column_name", "string", {"size": 50})`
* `null` - By default columns are not allowed to be `null`.
* `default` - The default value you want for this column. By default this is `null`.
* `default_raw` - The default value defined as a database function.
* `after` - (MySQL Only) Add a column after another column in the table. `example: {"after":"created_at"}`
* `first` - (MySQL Only) Add a column to the first position in the table. `example: {"first": true}`

#### Composite primary key 

```javascript
create_table("user_privileges") {
	t.Column("user_id", "int")
	t.Column("privilege_id", "int")
	t.PrimaryKey("user_id", "privilege_id")
}
```

Please note that the `t.PrimaryKey` statement MUST be after the columns definitions.

### Drop a Table

``` javascript
drop_table("table_name")
```

### Rename a Table

``` javascript
rename_table("old_table_name", "new_table_name")
```

### Add a Column

``` javascript
add_column("table_name", "column_name", "string", {})
```

See [above](#column-info) for more details on column types and options.

### Alter a column

``` javascript
change_column("table_name", "column_name", "string", {})
```

### Rename a Column

``` javascript
rename_column("table_name", "old_column_name", "new_column_name")
```

### Drop a Column

``` javascript
drop_column("table_name", "column_name")
```

### Add an Index

#### Supported Options:

* `name` - This defaults to `table_name_column_name_idx`
* `unique`

#### Simple Index:

``` javascript
add_index("table_name", "column_name", {})
```

#### Multi-Column Index:

``` javascript
add_index("table_name", ["column_1", "column_2"], {})
```

#### Unique Index:

``` javascript
add_index("table_name", "column_name", {"unique": true})
```

#### Index Names:

``` javascript
add_index("table_name", "column_name", {}) # name => table_name_column_name_idx
add_index("table_name", "column_name", {"name": "custom_index_name"})
```

### Rename an Index

``` javascript
rename_index("table_name", "old_index_name", "new_index_name")
```

### Drop an Index

``` javascript
drop_index("table_name", "index_name")
```

### Add a Foreign Key

```javascript
add_foreign_key("table_name", "field", {"ref_table_name": ["ref_column"]}, {
    "name": "optional_fk_name",
    "on_delete": "action",
    "on_update": "action",
})

```

#### Supported Options

* `name` - This defaults to `table_name_ref_table_name_ref_column_name_fk`
* `on_delete` - `CASCADE`, `SET NULL`, ...
* `on_update`

**Note:** `on_update` and `on_delete` are not supported on CockroachDB yet.

### Drop a Foreign Key

```javascript
drop_foreign_key("table_name", "fk_name", {"if_exists": true})
```

#### Supported Options

* `if_exists` - Adds `IF EXISTS` condition


### Raw SQL

``` javascript
sql("select * from users;")
```

### Execute an External Command

Sometimes during a migration you need to shell out to an external command.

```javascript
exec("echo hello")
```

## Development

### Testing

To run end-to-end tests, use

```
make test
```

If you made changes to the end-to-end tests and want to update the fixtures,
run the following command a couple of times until tests pass:

```
REFRESH_FIXTURES=true make test
```
