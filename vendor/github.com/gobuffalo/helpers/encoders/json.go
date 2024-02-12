package encoders

import (
	"encoding/json"
	"html/template"
)

// ToJSON marshals the interface{} and returns it
// as template.HTML
func ToJSON(v interface{}) (template.HTML, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.HTML(b), nil
}
