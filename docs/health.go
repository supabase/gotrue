package docs

// swagger:route GET /health config health
//
// Returns info about API version.
//
// responses:
//   200: healthResponse

// Current information about service version, name and extra.
// swagger:response healthResponse
type healthStatusResponse struct {
	// in:body
	Body struct {
		Version     string `json:"version"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
}
