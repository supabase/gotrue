package docs

import (
	"github.com/netlify/gotrue/api"
)

// swagger:route POST /recover recover recoverPostParams
//
// Start password recovery
//
// Will deliver a password recovery mail to the user based on email address.
//
// Security:
//  api_key:
// responses:
//   200: recoverPostResponse
//   401: httpErrorResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   422: httpErrorResponse
//   500: httpErrorResponse

// User's email who needs password recovery
// swagger:parameters recoverPostParams
type recoverPostParams struct {
	// in:body
	Body api.RecoverParams
}

// Empty json object
// swagger:response recoverPostResponse
type recoverPostResponse struct {
	// in:body
	Body struct{}
}
