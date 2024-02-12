package tags

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
)

// Options is a map used to configure tags
type Options map[string]interface{}

func (o Options) String() string {
	var out = make([]string, 0, len(o))
	for k, v := range o {
		if m, ok := v.(map[string]interface{}); ok {
			for mk, mv := range m {
				out = append(out, kv(fmt.Sprintf("%s-%s", k, mk), mv))
			}
			continue
		}
		out = append(out, kv(k, v))
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

func kv(k string, v interface{}) string {
	var tmp = make([]string, 2)
	tmp[0] = template.HTMLEscaper(k)
	if v != nil {
		tmp[1] = fmt.Sprintf("\"%s\"", template.HTMLEscaper(v))
		return strings.Join(tmp, "=")
	}
	// nil attribute value is interpreted as empty attribute notation
	// https://www.w3.org/TR/html5/syntax.html#elements-attributes
	return tmp[0]
}
