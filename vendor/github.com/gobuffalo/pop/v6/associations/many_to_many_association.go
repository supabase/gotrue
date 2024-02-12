package associations

import (
	"fmt"
	"reflect"
	"time"

	"github.com/gobuffalo/flect"
	"github.com/gobuffalo/pop/v6/internal/defaults"
	"github.com/gofrs/uuid"
)

type manyToManyAssociation struct {
	fieldType           reflect.Type
	fieldValue          reflect.Value
	model               reflect.Value
	manyToManyTableName string
	owner               interface{}
	fkID                string
	orderBy             string
	primaryID           string
	*associationSkipable
	*associationComposite
}

func init() {
	associationBuilders["many_to_many"] = func(p associationParams) (Association, error) {
		// Validates if model.ID is nil, this association will be skipped.
		var skipped bool
		model := p.modelValue
		if fieldIsNil(model.FieldByName("ID")) {
			skipped = true
		}

		return &manyToManyAssociation{
			fieldType:           p.modelValue.FieldByName(p.field.Name).Type(),
			fieldValue:          p.modelValue.FieldByName(p.field.Name),
			owner:               p.model,
			model:               model,
			manyToManyTableName: p.popTags.Find("many_to_many").Value,
			fkID:                p.popTags.Find("fk_id").Value,
			orderBy:             p.popTags.Find("order_by").Value,
			primaryID:           p.popTags.Find("primary_id").Value,
			associationSkipable: &associationSkipable{
				skipped: skipped,
			},
			associationComposite: &associationComposite{innerAssociations: p.innerAssociations},
		}, nil
	}
}

func (m *manyToManyAssociation) Kind() reflect.Kind {
	return m.fieldType.Kind()
}

func (m *manyToManyAssociation) Interface() interface{} {
	val := reflect.New(m.fieldType.Elem())
	if m.fieldValue.Kind() == reflect.Ptr {
		m.fieldValue.Set(val)
		return m.fieldValue.Interface()
	}

	// This piece of code clears a slice in case it is filled with elements.
	if m.fieldValue.Kind() == reflect.Slice || m.fieldValue.Kind() == reflect.Array {
		valPointer := m.fieldValue.Addr()
		valPointer.Elem().Set(reflect.MakeSlice(valPointer.Type().Elem(), 0, valPointer.Elem().Cap()))
		return valPointer.Interface()
	}

	return m.fieldValue.Addr().Interface()
}

// Constraint returns the content for a where clause, and the args
// needed to execute it.
func (m *manyToManyAssociation) Constraint() (string, []interface{}) {
	modelColumnID := defaults.String(m.primaryID, fmt.Sprintf("%s%s", flect.Underscore(m.model.Type().Name()), "_id"))

	var columnFieldID string
	i := reflect.Indirect(m.fieldValue)
	t := i.Type()
	if i.Kind() == reflect.Slice || i.Kind() == reflect.Array {
		t = t.Elem()
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	columnFieldID = defaults.String(m.fkID, fmt.Sprintf("%s%s", flect.Underscore(t.Name()), "_id"))

	subQuery := fmt.Sprintf("select %s from %s where %s = ?", columnFieldID, m.manyToManyTableName, modelColumnID)
	modelIDValue := m.model.FieldByName("ID").Interface()

	return fmt.Sprintf("id in (%s)", subQuery), []interface{}{modelIDValue}
}

func (m *manyToManyAssociation) OrderBy() string {
	return m.orderBy
}

func (m *manyToManyAssociation) BeforeInterface() interface{} {
	if m.fieldValue.Kind() == reflect.Ptr {
		return m.fieldValue.Interface()
	}
	return m.fieldValue.Addr().Interface()
}

func (m *manyToManyAssociation) BeforeSetup() error {
	return nil
}

func (m *manyToManyAssociation) Statements() []AssociationStatement {
	var statements []AssociationStatement

	modelColumnID := fmt.Sprintf("%s%s", flect.Underscore(m.model.Type().Name()), "_id")
	var columnFieldID string
	i := reflect.Indirect(m.fieldValue)
	if i.Kind() == reflect.Slice || i.Kind() == reflect.Array {
		t := i.Type().Elem()
		columnFieldID = fmt.Sprintf("%s%s", flect.Underscore(t.Name()), "_id")
	} else {
		columnFieldID = fmt.Sprintf("%s%s", flect.Underscore(i.Type().Name()), "_id")
	}

	for i := 0; i < m.fieldValue.Len(); i++ {
		v := m.fieldValue.Index(i)
		manyIDValue := v.FieldByName("ID").Interface()
		modelIDValue := m.model.FieldByName("ID").Interface()
		stm := "INSERT INTO %s (%s,%s,%s,%s) SELECT ?,?,?,? WHERE NOT EXISTS (SELECT * FROM %s WHERE %s = ? AND %s = ?)"

		if IsZeroOfUnderlyingType(manyIDValue) || IsZeroOfUnderlyingType(modelIDValue) {
			continue
		}

		associationStm := AssociationStatement{
			Statement: fmt.Sprintf(stm, m.manyToManyTableName, modelColumnID, columnFieldID, "created_at", "updated_at", m.manyToManyTableName, modelColumnID, columnFieldID),
			Args:      []interface{}{modelIDValue, manyIDValue, time.Now(), time.Now(), modelIDValue, manyIDValue},
		}

		if m.model.FieldByName("ID").Type().Name() == "UUID" {
			stm = "INSERT INTO %s (%s,%s,%s,%s,%s) SELECT ?,?,?,?,? WHERE NOT EXISTS (SELECT * FROM %s WHERE %s = ? AND %s = ?)"
			id, _ := uuid.NewV4()
			associationStm = AssociationStatement{
				Statement: fmt.Sprintf(stm, m.manyToManyTableName, "id", modelColumnID, columnFieldID, "created_at", "updated_at", m.manyToManyTableName, modelColumnID, columnFieldID),
				Args:      []interface{}{id, modelIDValue, manyIDValue, time.Now(), time.Now(), modelIDValue, manyIDValue},
			}
		}

		statements = append(statements, associationStm)
	}

	return statements
}
