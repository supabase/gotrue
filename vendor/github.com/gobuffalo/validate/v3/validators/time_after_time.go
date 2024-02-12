package validators

import (
	"fmt"
	"time"

	"github.com/gobuffalo/validate/v3"
)

type TimeAfterTime struct {
	FirstName  string
	FirstTime  time.Time
	SecondName string
	SecondTime time.Time
	Message    string
}

// IsValid adds an error if the FirstTime is not after the SecondTime.
func (v *TimeAfterTime) IsValid(errors *validate.Errors) {

	// UnixNano wraps around to negative numbers when a time is too far
	// into the future (e.g. 260 years)
	if v.FirstTime.Year() > v.SecondTime.Year() {
		return 
	}

	if v.FirstTime.UnixNano() >= v.SecondTime.UnixNano() {
		return
	}

	if len(v.Message) > 0 {
		errors.Add(GenerateKey(v.FirstName), v.Message)
		return
	}

	errors.Add(GenerateKey(v.FirstName), fmt.Sprintf("%s must be after %s.", v.FirstName, v.SecondName))
}
