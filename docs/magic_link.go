package docs

import (
	"github.com/netlify/gotrue/api"
)

// swagger:route POST /magiclink recover magiclinkPostParams
//
// Deliver link to user to redeem an access_token
//
// Magic Link will deliver a link (e.g. `/verify?type=recovery&token=fgtyuf68ddqdaDd`) to the user based on email address which they can use to redeem an access_token.
//
// responses:
//   200: recoverPostResponse
//   401: httpErrorResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   422: httpErrorResponse
//   500: httpErrorResponse

// User's email who needs password recovery
// swagger:parameters magiclinkPostParams
type magiclinkPostParams struct {
	// in:body
	Body api.MagicLinkParams
}

// Empty json object
// swagger:response magiclinkPostResponse
type magiclinkPostResponse struct {
	// in:body
	Body struct{}
}
