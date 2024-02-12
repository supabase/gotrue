package validators

import (
	"fmt"

	"github.com/gobuffalo/validate/v3"
)

// IntsAreEqual is a validator that will compare two integers and add
// an error if they are not equal
type IntsAreEqual struct {
	ValueOne int
	ValueTwo int
	Name     string
	Message  string
}

func (v *IntsAreEqual) IsValid(errors *validate.Errors) {
	if v.ValueOne != v.ValueTwo {
		errors.Add(GenerateKey(v.Name), fmt.Sprintf("%d is not equal to %d", v.ValueOne, v.ValueTwo))
	}
}
