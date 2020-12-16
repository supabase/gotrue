package docs

import "github.com/netlify/gotrue/api"

// swagger:route GET /verify recover verifyGetParams
//
// Verify a registration or a password recovery with redirect.
//
// Verify a registration or a password recovery and then redirect to app. Type can be `signup` or `recovery` and the `token` is a token returned from either `/signup` or `/recover`.
//
// responses:
//   301: verifyGetResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   422: httpErrorResponse
//   500: httpErrorResponse

// swagger:parameters verifyGetParams
type verifyGetParams struct {
	// specific verification type
	//
	// in:query
	// required: true
	// enum: signup, recovery
	Type string `json:"type"`

	// user's token
	//
	// in:query
	// required: true
	Token string `json:"token"`

	// set only when user invited and type is signup
	//
	// in:query
	Password string `json:"password"`
}

// Loged in and redirected to user's app with fields
// swagger:response verifyGetResponse
type verifyGetResponse struct {
	// in: query
	// required: true
	Token string `json:"access_token"`

	// in: query
	// required: true
	TokenType string `json:"token_type"`

	// in: query
	// required: true
	ExpiresIn int `json:"expires_in"`

	// in: query
	// required: true
	RefreshToken string `json:"refresh_token"`
}

// swagger:route POST /verify recover verifyPostParams
//
// Verify a registration or a password recovery.
//
// Verify a registration or a password recovery. Type can be `signup` or `recovery` and the `token` is a token returned from either `/signup` or `/recover`.
//
// responses:
//   200: verifyPostResponse
//   400: httpErrorResponse
//   401: httpErrorResponse
//   404: httpErrorResponse
//   422: httpErrorResponse
//   500: httpErrorResponse

// swagger:parameters verifyPostParams
type verifyPostParams struct {
	// All permited to change fields
	// in: body
	Body api.UserUpdateParams
}

// Successfully updated user info
// swagger:response verifyPostResponse
type verifyPostResponse struct {
	// swagger:allOf
	Body struct {
		Token        string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		User
	}
}
