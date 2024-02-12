package tags

import "github.com/gobuffalo/helpers/hctx"

// Keys to be used in templates for the functions in this package.
const (
	ImgKey          = "imgTag"
	CSSKey          = "stylesheetTag"
	JSKey           = "javascriptTag"
	LinkToKey       = "linkTo"
	RemoteLinkToKey = "remoteLinkTo"
	BackURLKey      = "backURL"
)

// New returns a map of the helpers within this package.
func New() hctx.Map {
	return hctx.Map{
		ImgKey:          Img,
		"img":           Img,
		"css":           CSS,
		"cssTag":        CSS,
		CSSKey:          CSS,
		"js":            JS,
		"jsTag":         JS,
		JSKey:           JS,
		LinkToKey:       LinkTo,
		RemoteLinkToKey: RemoteLinkTo,
		BackURLKey:      BackURL,
	}
}
