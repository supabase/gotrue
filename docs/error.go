package docs

// Occuered error description
// swagger:response httpErrorResponse
type httpErrorResponse struct {
	// in:body
	Body struct {
		Code    int    `json:"code"`
		Message string `json:"msg"`
		ErrorID string `json:"error_id,omitempty"`
	}
}
