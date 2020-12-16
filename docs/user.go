package docs

import "github.com/netlify/gotrue/api"

// swagger:route GET /user user userinfo
//
// Get the JSON object for the logged in user.
//
// Security:
//  api_key:
// responses:
//   200: userGetResponse
//   400: httpErrorResponse
//   404: httpErrorResponse
//   500: httpErrorResponse

// Get the JSON object for the logged in user.
// swagger:response userGetResponse
type userGetResponse struct {
	// swagger:allOf
	Body struct {
		User
		UserConfirms
	}
}

// swagger:model User
type User struct {
	ID        string            `json:"id"`
	Aud       string            `json:"aud"`
	Role      string            `json:"role"`
	Email     string            `json:"email"`
	AppMeta   map[string]string `json:"app_metadata"`
	UserMeta  map[string]string `json:"user_metadata"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// swagger:model UserConfirms
type UserConfirms struct {
	ConfirmedAt        string `json:"confirmed_at"`
	ConfirmationSentAt string `json:"confirmation_sent_at"`
	RecoverySentAt     string `json:"recovery_sent_at"`
	LastSignInAt       string `json:"last_sign_in_at"`
}

// swagger:route PUT /user user userupdate
//
// Update user's parameters.
//
// Apart from changing email/password, this method can be used to set custom user data.
//
// Security:
//  api_key:
// responses:
//   200: userGetResponse
//   400: httpErrorResponse
//   401: httpErrorResponse
//   404: httpErrorResponse
//   422: httpErrorResponse
//   500: httpErrorResponse

// swagger:parameters userupdate
type userUpdateParam struct {
	// All permited to change fields
	// in: body
	Body api.UserUpdateParams
}
