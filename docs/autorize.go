package docs

// swagger:route GET /authorize oauth authorizeGetParams
//
// Get access_token from external oauth provider.
//
// Start procedure of getting access_token by calling of external oauth provider and then to `/callback`.
//
// responses:
//   302: authorizeGetResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   500: httpErrorResponse

// swagger:parameters authorizeGetParams
type authorizeGetParams struct {
	// Provider type
	//
	// in:query
	// required: true
	// enum: bitbucket, github, gitlab, google, facebook, saml
	Type string `json:"type"`
}

// User redirected to external source
// swagger:response authorizeGetResponse
type authorizeGetResponse struct {
}
