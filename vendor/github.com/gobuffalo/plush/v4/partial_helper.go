package plush

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/github_flavored_markdown"
)

// PartialFeeder is callback function should implemented on application side.
type PartialFeeder func(string) (string, error)

func partialHelper(name string, data map[string]interface{}, help HelperContext) (template.HTML, error) {
	if help.Context == nil {
		return "", fmt.Errorf("invalid context. abort")
	}

	help.Context = help.New()
	for k, v := range data {
		help.Set(k, v)
	}

	pf, ok := help.Value("partialFeeder").(func(string) (string, error))
	if !ok {
		return "", fmt.Errorf("could not found partial feeder from helpers")
	}

	var part string
	var err error
	if part, err = pf(name); err != nil {
		return "", err
	}

	if part, err = Render(part, help.Context); err != nil {
		return "", err
	}

	if strings.HasSuffix(name, ".md") {
		part = string(github_flavored_markdown.Markdown([]byte(part)))
		part = strings.TrimSuffix(part, "\n")
	}

	if ct, ok := help.Value("contentType").(string); ok {
		ext := filepath.Ext(name)
		if strings.Contains(ct, "javascript") && ext != ".js" && ext != "" {
			part = template.JSEscapeString(string(part))
		}
	}

	if layout, ok := data["layout"].(string); ok {
		return partialHelper(
			layout,
			map[string]interface{}{"yield": template.HTML(part)},
			help)
	}

	return template.HTML(part), err
}
