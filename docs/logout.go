package docs

// swagger:route POST /logout user logoutPostParams
//
// Logout a user.
//
// This will revoke all refresh tokens for the user. Remember that the JWT tokens will still be valid for stateless auth until they expires.
//
// Security:
//  api_key:
//  responses:
//   204: logoutPostResponse
//   401: httpErrorResponse
//   400: httpErrorResponse
//   500: httpErrorResponse

// Empty logout params.
// swagger:parameters logoutPostParams
type logoutPostParams struct {
	// in:body
	Body struct {
	}
}

// Logout response.
// swagger:response logoutPostResponse
type logoutPostResponse struct {
	// in: body
	Body struct {
	}
}
