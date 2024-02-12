package pop

import (
	"fmt"
	"reflect"
	"time"

	"github.com/gobuffalo/pop/v6/associations"
	"github.com/gobuffalo/pop/v6/columns"
	"github.com/gobuffalo/pop/v6/logging"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// Reload fetch fresh data for a given model, using its ID.
func (c *Connection) Reload(model interface{}) error {
	sm := NewModel(model, c.Context())
	return sm.iterate(func(m *Model) error {
		return c.Find(m.Value, m.ID())
	})
}

// TODO: consider merging the following two methods.

// Exec runs the given query.
func (q *Query) Exec() error {
	return q.Connection.timeFunc("Exec", func() error {
		sql, args := q.ToSQL(nil)
		if sql == "" {
			return fmt.Errorf("empty query")
		}

		txlog(logging.SQL, q.Connection, sql, args...)
		_, err := q.Connection.Store.Exec(sql, args...)
		return err
	})
}

// ExecWithCount runs the given query, and returns the amount of
// affected rows.
func (q *Query) ExecWithCount() (int, error) {
	count := int64(0)
	return int(count), q.Connection.timeFunc("Exec", func() error {
		sql, args := q.ToSQL(nil)
		if sql == "" {
			return fmt.Errorf("empty query")
		}

		txlog(logging.SQL, q.Connection, sql, args...)
		result, err := q.Connection.Store.Exec(sql, args...)
		if err != nil {
			return err
		}

		count, err = result.RowsAffected()
		return err
	})
}

// ValidateAndSave applies validation rules on the given entry, then save it
// if the validation succeed, excluding the given columns.
//
// If model is a slice, each item of the slice is validated then saved in the database.
func (c *Connection) ValidateAndSave(model interface{}, excludeColumns ...string) (*validate.Errors, error) {
	sm := NewModel(model, c.Context())
	if err := sm.beforeValidate(c); err != nil {
		return nil, err
	}
	verrs, err := sm.validateSave(c)
	if err != nil {
		return verrs, err
	}
	if verrs.HasAny() {
		return verrs, nil
	}
	return verrs, c.Save(model, excludeColumns...)
}

var emptyUUID = uuid.Nil.String()

// IsZeroOfUnderlyingType will check if the value of anything is the equal to the Zero value of that type.
func IsZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

// Save wraps the Create and Update methods. It executes a Create if no ID is provided with the entry;
// or issues an Update otherwise.
//
// If model is a slice, each item of the slice is saved in the database.
func (c *Connection) Save(model interface{}, excludeColumns ...string) error {
	sm := NewModel(model, c.Context())
	return sm.iterate(func(m *Model) error {
		id, err := m.fieldByName("ID")
		if err != nil {
			return err
		}
		if IsZeroOfUnderlyingType(id.Interface()) {
			return c.Create(m.Value, excludeColumns...)
		}
		return c.Update(m.Value, excludeColumns...)
	})
}

// ValidateAndCreate applies validation rules on the given entry, then creates it
// if the validation succeed, excluding the given columns.
//
// If model is a slice, each item of the slice is validated then created in the database.
func (c *Connection) ValidateAndCreate(model interface{}, excludeColumns ...string) (*validate.Errors, error) {
	sm := NewModel(model, c.Context())

	isEager := c.eager
	hasEagerFields := c.eagerFields

	if err := sm.beforeValidate(c); err != nil {
		return nil, err
	}
	verrs, err := sm.validateCreate(c)
	if err != nil {
		return verrs, err
	}
	if verrs.HasAny() {
		return verrs, nil
	}

	if c.eager {
		asos, err := associations.ForStruct(model, c.eagerFields...)
		if err != nil {
			return verrs, fmt.Errorf("could not retrieve associations: %w", err)
		}

		if len(asos) == 0 {
			log(logging.Debug, "no associations found for given struct, disable eager mode")
			c.disableEager()
			return verrs, c.Create(model, excludeColumns...)
		}

		before := asos.AssociationsBeforeCreatable()
		for index := range before {
			i := before[index].BeforeInterface()
			if i == nil {
				continue
			}

			sm := NewModel(i, c.Context())
			verrs, err := sm.validateAndOnlyCreate(c)
			if err != nil || verrs.HasAny() {
				return verrs, err
			}
		}

		after := asos.AssociationsAfterCreatable()
		for index := range after {
			i := after[index].AfterInterface()
			if i == nil {
				continue
			}

			sm := NewModel(i, c.Context())
			verrs, err := sm.validateAndOnlyCreate(c)
			if err != nil || verrs.HasAny() {
				return verrs, err
			}
		}

		sm := NewModel(model, c.Context())
		verrs, err = sm.validateCreate(c)
		if err != nil || verrs.HasAny() {
			return verrs, err
		}
	}

	c.eager = isEager
	c.eagerFields = hasEagerFields
	return verrs, c.Create(model, excludeColumns...)
}

// Create add a new given entry to the database, excluding the given columns.
// It updates `created_at` and `updated_at` columns automatically.
//
// If model is a slice, each item of the slice is created in the database.
//
// Create support two modes:
// * Flat (default): Associate existing nested objects only. NO creation or update of nested objects.
// * Eager: Associate existing nested objects and create non-existent objects. NO change to existing objects.
func (c *Connection) Create(model interface{}, excludeColumns ...string) error {
	var isEager = c.eager

	c.disableEager()

	sm := NewModel(model, c.Context())
	return sm.iterate(func(m *Model) error {
		return c.timeFunc("Create", func() error {
			var localIsEager = isEager
			asos, err := associations.ForStruct(m.Value, c.eagerFields...)
			if err != nil {
				return fmt.Errorf("could not retrieve associations: %w", err)
			}

			if localIsEager && len(asos) == 0 {
				// No association, fallback to non-eager mode.
				localIsEager = false
			}

			if err = m.beforeSave(c); err != nil {
				return err
			}

			if err = m.beforeCreate(c); err != nil {
				return err
			}

			processAssoc := len(asos) > 0

			if processAssoc {
				before := asos.AssociationsBeforeCreatable()
				for index := range before {
					i := before[index].BeforeInterface()
					if i == nil {
						continue
					}

					if localIsEager {
						sm := NewModel(i, c.Context())
						err = sm.iterate(func(m *Model) error {
							id, err := m.fieldByName("ID")
							if err != nil {
								return err
							}
							if IsZeroOfUnderlyingType(id.Interface()) {
								return c.Create(m.Value)
							}
							return nil
						})

						if err != nil {
							return err
						}
					}

					err = before[index].BeforeSetup()
					if err != nil {
						return err
					}
				}
			}

			tn := m.TableName()
			cols := m.Columns()

			if tn == sm.TableName() {
				cols.Remove(excludeColumns...)
			}

			now := nowFunc().Truncate(time.Microsecond)
			m.setUpdatedAt(now)
			m.setCreatedAt(now)

			if err = c.Dialect.Create(c, m, cols); err != nil {
				return err
			}

			if processAssoc {
				after := asos.AssociationsAfterCreatable()
				for index := range after {
					if localIsEager {
						err = after[index].AfterSetup()
						if err != nil {
							return err
						}

						i := after[index].AfterInterface()
						if i == nil {
							continue
						}

						sm := NewModel(i, c.Context())
						err = sm.iterate(func(m *Model) error {
							fbn, err := m.fieldByName("ID")
							if err != nil {
								return err
							}
							id := fbn.Interface()
							if IsZeroOfUnderlyingType(id) {
								return c.Create(m.Value)
							}

							exists, errE := Q(c).Where(m.WhereID(), id).Exists(i)
							if errE != nil || !exists {
								return c.Create(m.Value)
							}
							return nil
						})

						if err != nil {
							return err
						}
					}
					stm := after[index].AfterProcess()
					if c.TX != nil && !stm.Empty() {
						err := c.RawQuery(c.Dialect.TranslateSQL(stm.Statement), stm.Args...).Exec()
						if err != nil {
							return err
						}
					}
				}

				stms := asos.AssociationsCreatableStatement()
				for index := range stms {
					statements := stms[index].Statements()
					for _, stm := range statements {
						err := c.RawQuery(c.Dialect.TranslateSQL(stm.Statement), stm.Args...).Exec()
						if err != nil {
							return err
						}
					}
				}
			}

			if err = m.afterCreate(c); err != nil {
				return err
			}

			return m.afterSave(c)
		})
	})
}

// ValidateAndUpdate applies validation rules on the given entry, then update it
// if the validation succeed, excluding the given columns.
//
// If model is a slice, each item of the slice is validated then updated in the database.
func (c *Connection) ValidateAndUpdate(model interface{}, excludeColumns ...string) (*validate.Errors, error) {
	sm := NewModel(model, c.Context())
	if err := sm.beforeValidate(c); err != nil {
		return nil, err
	}
	verrs, err := sm.validateUpdate(c)
	if err != nil {
		return verrs, err
	}
	if verrs.HasAny() {
		return verrs, nil
	}
	return verrs, c.Update(model, excludeColumns...)
}

// Update writes changes from an entry to the database, excluding the given columns.
// It updates the `updated_at` column automatically.
//
// If model is a slice, each item of the slice is updated in the database.
func (c *Connection) Update(model interface{}, excludeColumns ...string) error {
	sm := NewModel(model, c.Context())
	return sm.iterate(func(m *Model) error {
		return c.timeFunc("Update", func() error {
			var err error

			if err = m.beforeSave(c); err != nil {
				return err
			}
			if err = m.beforeUpdate(c); err != nil {
				return err
			}

			tn := m.TableName()
			cols := columns.ForStructWithAlias(model, tn, m.As, m.IDField())
			cols.Remove(m.IDField(), "created_at")

			if tn == sm.TableName() {
				cols.Remove(excludeColumns...)
			}

			now := nowFunc().Truncate(time.Microsecond)
			m.setUpdatedAt(now)

			if err = c.Dialect.Update(c, m, cols); err != nil {
				return err
			}
			if err = m.afterUpdate(c); err != nil {
				return err
			}

			return m.afterSave(c)
		})
	})
}

// UpdateQuery updates all rows matched by the query. The new values are read
// from the first argument, which must be a struct. The column names to be
// updated must be listed explicitly in subsequent arguments. The ID and
// CreatedAt columns are never updated. The UpdatedAt column is updated
// automatically.
//
// UpdateQuery does not execute (before|after)(Create|Update|Save) callbacks.
//
// Calling UpdateQuery with no columnNames will result in only the UpdatedAt
// column being updated.
func (q *Query) UpdateQuery(model interface{}, columnNames ...string) (int64, error) {
	sm := NewModel(model, q.Connection.Context())
	modelKind := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(model))).Kind()
	if modelKind != reflect.Struct {
		return 0, fmt.Errorf("model must be a struct; got %s", modelKind)
	}

	cols := columns.NewColumnsWithAlias(sm.TableName(), sm.As, sm.IDField())
	cols.Add(columnNames...)
	if _, err := sm.fieldByName("UpdatedAt"); err == nil {
		cols.Add("updated_at")
	}
	cols.Remove(sm.IDField(), "created_at")

	now := nowFunc().Truncate(time.Microsecond)
	sm.setUpdatedAt(now)
	return q.Connection.Dialect.UpdateQuery(q.Connection, sm, cols, *q)
}

// UpdateColumns writes changes from an entry to the database, including only the given columns
// or all columns if no column names are provided.
// It updates the `updated_at` column automatically if you include `updated_at` in columnNames.
//
// If model is a slice, each item of the slice is updated in the database.
func (c *Connection) UpdateColumns(model interface{}, columnNames ...string) error {
	sm := NewModel(model, c.Context())
	return sm.iterate(func(m *Model) error {
		return c.timeFunc("Update", func() error {
			var err error

			if err = m.beforeSave(c); err != nil {
				return err
			}
			if err = m.beforeUpdate(c); err != nil {
				return err
			}

			tn := m.TableName()

			cols := columns.Columns{}
			if len(columnNames) > 0 && tn == sm.TableName() {
				cols = columns.NewColumnsWithAlias(tn, m.As, sm.IDField())
				cols.Add(columnNames...)

			} else {
				cols = columns.ForStructWithAlias(model, tn, m.As, m.IDField())
			}
			cols.Remove("id", "created_at")

			now := nowFunc().Truncate(time.Microsecond)
			m.setUpdatedAt(now)

			if err = c.Dialect.Update(c, m, cols); err != nil {
				return err
			}
			if err = m.afterUpdate(c); err != nil {
				return err
			}

			return m.afterSave(c)
		})
	})
}

// Destroy deletes a given entry from the database.
//
// If model is a slice, each item of the slice is deleted from the database.
func (c *Connection) Destroy(model interface{}) error {
	sm := NewModel(model, c.Context())
	return sm.iterate(func(m *Model) error {
		return c.timeFunc("Destroy", func() error {
			var err error

			if err = m.beforeDestroy(c); err != nil {
				return err
			}
			if err = c.Dialect.Destroy(c, m); err != nil {
				return err
			}

			return m.afterDestroy(c)
		})
	})
}

func (q *Query) Delete(model interface{}) error {
	q.Operation = Delete

	return q.Connection.timeFunc("Delete", func() error {
		m := NewModel(model, q.Connection.Context())
		err := q.Connection.Dialect.Delete(q.Connection, m, *q)
		if err != nil {
			return err
		}
		return m.afterDestroy(q.Connection)
	})
}
