package docs

// swagger:route GET /callback oauth callbackGetParams
//
// External provider redirects to here
//
// Start procedure of creating token
//
// responses:
//   302: callbackGetResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   500: httpErrorResponse

// swagger:parameters callbackGetParams
type callbackGetParams struct {
	// Error message
	//
	// in:query
	Error string `json:"error"`

	// Auth code returned by provider
	//
	// in:query
	Code string `json:"code"`
}

// User redirected to GOTRUE_SITE_URL with query parameters splitted by #
// swagger:response authorizeGetResponse
type callbackGetResponse struct {
	// in: query
	// required: true
	AccessToken string `json:"access_token"`
	// in: query
	// required: true
	RefreshToken string `json:"refresh_token"`
	// in: query
	// required: true
	ExpiresIn string `json:"expires_in"`
	// in: query
	// required: true
	Provider string `json:"provider"`
}
