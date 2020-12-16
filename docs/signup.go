package docs

import (
	"github.com/netlify/gotrue/api"
)

// swagger:route POST /signup signup signupPostParams
//
// Register a new user with an email and password.
//
// responses:
//   200: signupPostResponse
//   401: httpErrorResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   500: httpErrorResponse

// Register a new user with an email and password.
// swagger:parameters signupPostParams
type signupPostParams struct {
	// in:body
	Body api.SignupParams
}

// Registered user information
// swagger:response signupPostResponse
type signupPostResponse struct {
	// in: body
	Body User
}
