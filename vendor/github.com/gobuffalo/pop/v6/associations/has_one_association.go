package associations

import (
	"fmt"
	"reflect"

	"github.com/gobuffalo/flect"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6/internal/defaults"
)

// hasOneAssociation is a 1 to 1 kind of association. It's used on
// the side the association foreign key is not defined.
//
// See the belongsToAssociation for the other side of the relation.
type hasOneAssociation struct {
	ownedTableName string
	ownedModel     reflect.Value
	ownedType      reflect.Type
	ownerID        interface{}
	ownerName      string
	owner          interface{}
	fkID           string
	*associationSkipable
	*associationComposite
}

func init() {
	associationBuilders["has_one"] = hasOneAssociationBuilder
}

func hasOneAssociationBuilder(p associationParams) (Association, error) {
	// Validates if ownerIDField is nil, this association will be skipped.
	var skipped bool
	ownerID := p.modelValue.FieldByName("ID")
	if fieldIsNil(ownerID) {
		skipped = true
	}

	ownerName := p.modelType.Name()
	fk := defaults.String(p.popTags.Find("fk_id").Value, flect.Underscore(ownerName)+"_id")

	fval := p.modelValue.FieldByName(p.field.Name)
	return &hasOneAssociation{
		owner:          p.model,
		ownedTableName: flect.Pluralize(p.popTags.Find("has_one").Value),
		ownedModel:     fval,
		ownedType:      fval.Type(),
		ownerID:        ownerID.Interface(),
		ownerName:      ownerName,
		fkID:           fk,
		associationSkipable: &associationSkipable{
			skipped: skipped,
		},
		associationComposite: &associationComposite{innerAssociations: p.innerAssociations},
	}, nil
}

func (h *hasOneAssociation) Kind() reflect.Kind {
	if h.ownedType.Kind() == reflect.Ptr {
		return h.ownedType.Elem().Kind()
	}
	return h.ownedType.Kind()
}

func (h *hasOneAssociation) Interface() interface{} {
	if h.ownedModel.Kind() == reflect.Ptr {
		val := reflect.New(h.ownedType.Elem())
		h.ownedModel.Set(val)
		return h.ownedModel.Interface()
	}
	return h.ownedModel.Addr().Interface()
}

// Constraint returns the content for the WHERE clause, and the args
// needed to execute it.
func (h *hasOneAssociation) Constraint() (string, []interface{}) {
	return fmt.Sprintf("%s = ?", h.fkID), []interface{}{h.ownerID}
}

func (h *hasOneAssociation) AfterSetup() error {
	om := h.ownedModel
	if fieldIsNil(om) {
		return nil
	}
	ownerID := reflect.Indirect(reflect.ValueOf(h.owner)).FieldByName("ID").Interface()
	if om.Kind() == reflect.Ptr {
		om = om.Elem()
	}
	fval := om.FieldByName(h.ownerName + "ID")
	if fval.CanSet() {
		if n := nulls.New(fval.Interface()); n != nil {
			fval.Set(reflect.ValueOf(n.Parse(ownerID)))
		} else {
			fval.Set(reflect.ValueOf(ownerID))
		}
		return nil
	}

	return fmt.Errorf("could not set '%s' to '%s'", ownerID, fval)
}

// AfterInterface gets the value of the model to create after
// creating the parent model. It returns nil if its value is
// not set.
func (h *hasOneAssociation) AfterInterface() interface{} {
	m := h.ownedModel
	if fieldIsNil(m) {
		return nil
	}
	if m.Kind() == reflect.Ptr {
		return m.Interface()
	}
	if IsZeroOfUnderlyingType(m.Interface()) {
		return nil
	}

	return m.Addr().Interface()
}

func (h *hasOneAssociation) AfterProcess() AssociationStatement {
	belongingIDFieldName := "ID"
	om := h.ownedModel
	if om.Kind() == reflect.Ptr {
		om = om.Elem()
	}
	// Skip if the related model is not set
	if IsZeroOfUnderlyingType(om) {
		return AssociationStatement{
			Statement: "",
			Args:      []interface{}{},
		}
	}
	id := om.FieldByName(belongingIDFieldName).Interface()
	if IsZeroOfUnderlyingType(id) {
		return AssociationStatement{
			Statement: "",
			Args:      []interface{}{},
		}
	}

	ownerIDFieldName := "ID"
	ownerID := reflect.Indirect(reflect.ValueOf(h.owner)).FieldByName(ownerIDFieldName).Interface()

	ids := []interface{}{ownerID}
	ids = append(ids, id)

	ret := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", h.ownedTableName, h.fkID, belongingIDFieldName)

	return AssociationStatement{
		Statement: ret,
		Args:      ids,
	}
}
