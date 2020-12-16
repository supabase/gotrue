package docs

// swagger:route POST /token token tokenPostParams
//
// This is an OAuth2 endpoint that currently implements the password and refresh_token grant types.
//
// This is an OAuth2 endpoint that currently implements the password and refresh_token grant types. Once you have an access token, you can access the methods requiring authentication by settings the `Authorization: Bearer YOUR_ACCESS_TOKEN_HERE` header.
//
// responses:
//   200: tokenPostResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   422: httpErrorResponse
//   500: httpErrorResponse

// swagger:parameters tokenPostParams
type tokenPostParams struct {
	// specific verification type
	//
	// in:query
	// required: true
	// enum: password, refresh_token
	Type string `json:"grant_type"`

	// Provided info by user. Email + password or refresh_token only needed
	// required: true
	// in: body
	Body struct {
		Email        string `json:"email"`
		Password     string `json:"password"`
		RefreshToken string `json:"refresh_token"`
	}
}

// Get new token or refresh the old one
// swagger:response tokenPostResponse
type tokenPostResponse struct {
	// swagger: allOf
	Body verifyGetResponse
}
