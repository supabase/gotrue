package associations

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gobuffalo/pop/v6/columns"
)

// If a field match with the regexp, it will be considered as a valid field definition.
// e.g: "MyField"             => valid.
// e.g: "MyField.NestedField" => valid.
// e.g: "MyField."            => not valid.
// e.g: "MyField.*"           => not valid for now.
var validAssociationExpRegexp = regexp.MustCompile(`^(([a-zA-Z0-9]*)(\.[a-zA-Z0-9]+)?)+$`)

// associationBuilders is a map that helps to aisle associations finding process
// with the associations implementation. Every association MUST register its builder
// in this map using its init() method. see ./has_many_association.go as a guide.
var associationBuilders = map[string]associationBuilder{}

// ForStruct returns all associations for
// the struct specified. It takes into account tags
// associations like has_many, belongs_to, has_one.
// it throws an error when it finds a field that does
// not exist for a model.
func ForStruct(s interface{}, fields ...string) (Associations, error) {
	return forStruct(s, s, fields)
}

// forStruct is a recursive helper that passes the root model down for embedded fields
func forStruct(parent, s interface{}, fields []string) (Associations, error) {
	t, v := getModelDefinition(s)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("could not get struct associations: not a struct but %T", s)
	}
	fields = trimFields(fields)
	associations := Associations{}
	fieldsWithInnerAssociation := map[string]InnerAssociations{}

	// validate if fields contains a non existing field in struct.
	// and verify is it has inner associations.
	for i := range fields {
		var innerField string

		if !validAssociationExpRegexp.MatchString(fields[i]) {
			return associations, fmt.Errorf("association '%s' does not match the format %s", fields[i], "'<field>' or '<field>.<nested-field>'")
		}

		fields[i], innerField = extractFieldAndInnerFields(fields[i])

		if _, ok := t.FieldByName(fields[i]); !ok {
			return associations, fmt.Errorf("field %s does not exist in model %s", fields[i], t.Name())
		}

		if innerField != "" {
			var found bool
			innerF, _ := extractFieldAndInnerFields(innerField)

			for j := range fieldsWithInnerAssociation[fields[i]] {
				f, _ := extractFieldAndInnerFields(fieldsWithInnerAssociation[fields[i]][j].Fields[0])
				if innerF == f {
					fieldsWithInnerAssociation[fields[i]][j].Fields = append(fieldsWithInnerAssociation[fields[i]][j].Fields, innerField)
					found = true
					break
				}
			}

			if !found {
				fieldsWithInnerAssociation[fields[i]] = append(fieldsWithInnerAssociation[fields[i]], InnerAssociation{fields[i], []string{innerField}})
			}
		}
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// inline embedded field
		if f.Anonymous {
			field := v.Field(i)
			// we need field to be a pointer, so that we can later set the value
			// if the embedded field is of type struct {...}, we have to take its address
			if field.Kind() != reflect.Ptr {
				field = field.Addr()
			}
			if fieldIsNil(field) {
				// initialize zero value
				field = reflect.New(field.Type().Elem())
				// we can only get in this case if v.Field(i) is a pointer type because it could not be nil otherwise
				//  => it is safe to set it here as is
				v.Field(i).Set(field)
			}
			innerAssociations, err := forStruct(parent, field.Interface(), fields)
			if err != nil {
				return nil, err
			}
			associations = append(associations, innerAssociations...)
			continue
		}

		// ignores those fields not included in fields list.
		if len(fields) > 0 && fieldIgnoredIn(fields, f.Name) {
			continue
		}

		tags := columns.TagsFor(f)

		for name, builder := range associationBuilders {
			tag := tags.Find(name)
			if !tag.Empty() {
				pt, pv := getModelDefinition(parent)
				params := associationParams{
					field:             f,
					model:             parent,
					modelType:         pt,
					modelValue:        pv,
					popTags:           tags,
					innerAssociations: fieldsWithInnerAssociation[f.Name],
				}

				a, err := builder(params)
				if err != nil {
					return associations, err
				}

				associations = append(associations, a)
				break
			}
		}
	}

	return associations, nil
}

func getModelDefinition(s interface{}) (reflect.Type, reflect.Value) {
	v := reflect.ValueOf(s)
	v = reflect.Indirect(v)
	t := v.Type()
	return t, v
}

func trimFields(fields []string) []string {
	var trimFields []string
	for _, f := range fields {
		if strings.TrimSpace(f) != "" {
			trimFields = append(trimFields, strings.TrimSpace(f))
		}
	}
	return trimFields
}

func fieldIgnoredIn(fields []string, field string) bool {
	for _, f := range fields {
		if f == field {
			return false
		}
	}
	return true
}

func extractFieldAndInnerFields(field string) (string, string) {
	if !strings.Contains(field, ".") {
		return field, ""
	}

	dotIndex := strings.Index(field, ".")
	return field[:dotIndex], field[dotIndex+1:]
}
