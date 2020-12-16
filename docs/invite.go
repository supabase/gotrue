package docs

import (
	"github.com/netlify/gotrue/api"
)

// swagger:route POST /invite recover invitePostParams
//
// Invite a new user by email (require admin privilege).
//
// Security:
//  api_key:
// responses:
//   200: invitePostResponse
//   401: httpErrorResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   500: httpErrorResponse

// Get the JSON object for the logged in user.
// swagger:parameters invitePostParams
type invitePostParams struct {
	// in:body
	Body api.InviteParams
}

// Invited user information
// swagger:response invitePostResponse
type invitePostResponse struct {
	// in: body
	Body User
}
