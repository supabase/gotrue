package validators

import (
	"fmt"
	"strings"

	"github.com/gobuffalo/validate/v3"
)

type FuncValidator struct {
	Fn      func() bool
	Field   string
	Name    string
	Message string
}

func (f *FuncValidator) IsValid(verrs *validate.Errors) {
	// for backwards compatability
	if strings.TrimSpace(f.Name) == "" {
		f.Name = f.Field
	}
	if !f.Fn() {
		verrs.Add(GenerateKey(f.Name), fmt.Sprintf(f.Message, f.Field))
	}
}
