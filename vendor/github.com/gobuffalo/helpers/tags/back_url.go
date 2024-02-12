package tags

import (
	"net/http"

	"github.com/gobuffalo/helpers/hctx"
)

// BackURL returns a URL to the referer, if its presend in the
// "Referer" header it will take it from there. Otherwise it will return
// "javascript:history.back()" and rely on the browser history.
func BackURL(help hctx.HelperContext) string {
	backURL := "javascript:history.back()"

	var req *http.Request
	var ok bool
	if req, ok = help.Value("request").(*http.Request); !ok {
		return backURL
	}

	if referer := req.Header.Get("Referer"); referer != "" {
		backURL = referer
	}

	return backURL
}
