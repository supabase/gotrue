package validators

import (
	"fmt"

	"github.com/gobuffalo/validate/v3"
)

// IntsAreNotEqual is a validator that compares two integers and will add
// an error if they are equal
type IntsAreNotEqual struct {
	ValueOne int
	ValueTwo int
	Name     string
	Message  string
}

func (v *IntsAreNotEqual) IsValid(errors *validate.Errors) {
	if v.ValueOne == v.ValueTwo {
		errors.Add(GenerateKey(v.Name), fmt.Sprintf("%d is equal to %d", v.ValueOne, v.ValueTwo))
	}
}
