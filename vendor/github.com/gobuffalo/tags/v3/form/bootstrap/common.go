package bootstrap

import (
	"fmt"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/gobuffalo/tags/v3"
)

func buildOptions(opts tags.Options, err bool) {
	if opts["class"] == nil {
		opts["class"] = ""
	}

	if opts["tag_only"] != true {
		if opts["type"] == "checkbox" {
			opts["class"] = strings.Join([]string{fmt.Sprint(opts["class"]), "form-check-input"}, " ")
		} else {
			opts["class"] = strings.Join([]string{fmt.Sprint(opts["class"]), "form-control"}, " ")
		}
	}

	if err {
		opts["class"] = strings.Join([]string{fmt.Sprint(opts["class"]), "is-invalid"}, " ")
	}

	opts["class"] = strings.TrimSpace(opts["class"].(string))
	delete(opts, "hide_label")
}

func divWrapper(opts tags.Options, fn func(opts tags.Options) tags.Body) *tags.Tag {
	divClass := "form-group" // btw, form-group was deprecated in Bootstrap 5
	labelClass := "form-label"
	hasErrors := false
	errors := []string{}
	hasHelp := false
	helpMessage := ""

	if opts["div_class"] != nil {
		divClass = opts["div_class"].(string)
		delete(opts, "div_class")
	}

	if opts["label_class"] != nil {
		labelClass = opts["label_class"].(string)
		delete(opts, "label_class")
	}

	if opts["bootstrap"] != nil {
		bopts, ok := opts["bootstrap"].(map[string]interface{})
		if ok {
			divClass = bopts["form-group-class"].(string)
		}

		delete(opts, "bootstrap")
	}

	if opts["help"] != nil {
		hasHelp = true
		helpMessage = opts["help"].(string)
		delete(opts, "help")
	}

	if opts["errors"] != nil && len(opts["errors"].([]string)) > 0 {
		divClass += " has-error"
		hasErrors = true
		errors = opts["errors"].([]string)
		delete(opts, "errors")
	}

	div := tags.New("div", tags.Options{
		"class": divClass,
	})

	if opts["label"] == nil && opts["tags-field"] != nil {
		if tf, ok := opts["tags-field"].(string); ok {
			tf = strings.Join(strings.Split(tf, "."), " ")
			opts["label"] = flect.Titleize(tf)
		}
	}
	delete(opts, "tags-field")

	useLabel := opts["hide_label"] == nil
	if useLabel && opts["label"] != nil {
		div.Prepend(tags.New("label", tags.Options{
			"for":   opts["id"],
			"body":  opts["label"],
			"class": labelClass,
		}))
		delete(opts, "label")
	}

	buildOptions(opts, hasErrors)

	if opts["tag_only"] == true {
		return fn(opts).(*tags.Tag)
	}

	isFloatingLabel := strings.Contains(divClass, "form-floating")
	if opts["type"] == "checkbox" || isFloatingLabel {
		if isFloatingLabel && opts["placeholder"] == nil {
			// bootstrap 5 floating label requires this
			opts["placeholder"] = opts["name"]
		}
		div.Prepend(fn(opts))
	} else {
		div.Append(fn(opts))
	}

	if hasErrors {
		for _, err := range errors {
			div.Append(tags.New("div", tags.Options{
				"class": "invalid-feedback help-block",
				"body":  err,
			}))
		}
	}

	if hasHelp {
		div.Append(tags.New("div", tags.Options{
			"id":    fmt.Sprintf("%v-help", opts["name"]),
			"class": "form-text",
			"body":  helpMessage,
		}))
	}

	return div
}
