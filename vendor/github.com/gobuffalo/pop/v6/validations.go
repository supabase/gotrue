package pop

import (
	"reflect"

	"github.com/gobuffalo/validate/v3"
)

type beforeValidatable interface {
	BeforeValidations(*Connection) error
}

type validateable interface {
	Validate(*Connection) (*validate.Errors, error)
}

type modelIterableValidator func(*Model) (*validate.Errors, error)

func (m *Model) validate(c *Connection) (*validate.Errors, error) {
	if x, ok := m.Value.(beforeValidatable); ok {
		if err := x.BeforeValidations(c); err != nil {
			return validate.NewErrors(), err
		}
	}
	if x, ok := m.Value.(validateable); ok {
		return x.Validate(c)
	}
	return validate.NewErrors(), nil
}

type validateCreateable interface {
	ValidateCreate(*Connection) (*validate.Errors, error)
}

func (m *Model) validateCreate(c *Connection) (*validate.Errors, error) {
	return m.iterateAndValidate(func(model *Model) (*validate.Errors, error) {
		verrs, err := model.validate(c)
		if err != nil {
			return verrs, err
		}
		if x, ok := model.Value.(validateCreateable); ok {
			vs, err := x.ValidateCreate(c)
			if vs != nil {
				verrs.Append(vs)
			}
			if err != nil {
				return verrs, err
			}
		}

		return verrs, err
	})
}

func (m *Model) validateAndOnlyCreate(c *Connection) (*validate.Errors, error) {
	return m.iterateAndValidate(func(model *Model) (*validate.Errors, error) {
		id, err := model.fieldByName("ID")
		if err != nil {
			return nil, err
		}
		if !IsZeroOfUnderlyingType(id.Interface()) {
			return validate.NewErrors(), nil
		}

		verrs, err := model.validate(c)
		if err != nil {
			return verrs, err
		}
		if x, ok := model.Value.(validateCreateable); ok {
			vs, err := x.ValidateCreate(c)
			if vs != nil {
				verrs.Append(vs)
			}
			if err != nil {
				return verrs, err
			}
		}

		return verrs, err
	})
}

type validateSaveable interface {
	ValidateSave(*Connection) (*validate.Errors, error)
}

func (m *Model) validateSave(c *Connection) (*validate.Errors, error) {
	return m.iterateAndValidate(func(model *Model) (*validate.Errors, error) {
		verrs, err := model.validate(c)
		if err != nil {
			return verrs, err
		}
		if x, ok := model.Value.(validateSaveable); ok {
			vs, err := x.ValidateSave(c)
			if vs != nil {
				verrs.Append(vs)
			}
			if err != nil {
				return verrs, err
			}
		}

		return verrs, err
	})
}

type validateUpdateable interface {
	ValidateUpdate(*Connection) (*validate.Errors, error)
}

func (m *Model) validateUpdate(c *Connection) (*validate.Errors, error) {
	return m.iterateAndValidate(func(model *Model) (*validate.Errors, error) {
		verrs, err := model.validate(c)
		if err != nil {
			return verrs, err
		}
		if x, ok := model.Value.(validateUpdateable); ok {
			vs, err := x.ValidateUpdate(c)
			if vs != nil {
				verrs.Append(vs)
			}
			if err != nil {
				return verrs, err
			}
		}

		return verrs, err
	})
}

func (m *Model) iterateAndValidate(fn modelIterableValidator) (*validate.Errors, error) {
	v := reflect.Indirect(reflect.ValueOf(m.Value))
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i)
			newModel := NewModel(val.Addr().Interface(), m.ctx)
			verrs, err := fn(newModel)

			if err != nil || verrs.HasAny() {
				return verrs, err
			}
		}
		return validate.NewErrors(), nil
	}

	return fn(m)
}
