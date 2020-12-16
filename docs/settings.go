package docs

import (
	"github.com/netlify/gotrue/api"
)

// swagger:route GET /settings config settings
//
// Returns the publicly available settings for this gotrue instance.
//
// responses:
//   200: settingsResponse

// Publicly available settings for this gotrue instance.
// swagger:response settingsResponse
type settingsResponse struct {
	// in:body
	Body api.Settings
}
