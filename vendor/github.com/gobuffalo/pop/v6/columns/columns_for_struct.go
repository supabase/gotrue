package columns

import (
	"reflect"
)

// ForStruct returns a Columns instance for
// the struct passed in.
func ForStruct(s interface{}, tableName, idField string) (columns Columns) {
	return ForStructWithAlias(s, tableName, "", idField)
}

// ForStructWithAlias returns a Columns instance for the struct passed in.
// If the tableAlias is not empty, it will be used.
func ForStructWithAlias(s interface{}, tableName, tableAlias, idField string) (columns Columns) {
	columns = NewColumnsWithAlias(tableName, tableAlias, idField)
	defer func() {
		if r := recover(); r != nil {
			columns = NewColumnsWithAlias(tableName, tableAlias, idField)
			columns.Add("*")
		}
	}()
	st := reflect.TypeOf(s)
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}
	if st.Kind() == reflect.Slice {
		st = st.Elem()
		if st.Kind() == reflect.Ptr {
			st = st.Elem()
		}
	}

	// recursive functions to also find and add embedded struct fields
	var findColumns func(st reflect.Type)
	findColumns = func(t reflect.Type) {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		fc := t.NumField()
		for i := 0; i < fc; i++ {
			field := t.Field(i)

			if field.Anonymous {
				findColumns(field.Type)
				continue
			}

			popTags := TagsFor(field)
			tag := popTags.Find("db")

			if !tag.Ignored() && !tag.Empty() {
				col := tag.Value

				// add writable or readable.
				tag := popTags.Find("rw")
				if !tag.Empty() {
					col = col + "," + tag.Value
				}

				cs := columns.Add(col)

				// add select clause.
				tag = popTags.Find("select")
				if !tag.Empty() {
					c := cs[0]
					c.SetSelectSQL(tag.Value)
				}
			}
		}
	}

	findColumns(st)

	return columns
}
