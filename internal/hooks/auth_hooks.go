package hooks

import (
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt"
	"github.com/supabase/gotrue/internal/models"
)

type HookType string

const (
	PostgresHook HookType = "pg-functions"
)

const (
	// In Miliseconds
	DefaultTimeout = 2000
)

// Hook Names
const (
	HookRejection = "reject"
)

type HookOutput interface {
	IsError() bool
	Error() string
}

// GoTrueClaims is a struct thats used for JWT claims
type GoTrueClaims struct {
	jwt.StandardClaims
	Email                         string                 `json:"email"`
	Phone                         string                 `json:"phone"`
	AppMetaData                   map[string]interface{} `json:"app_metadata"`
	UserMetaData                  map[string]interface{} `json:"user_metadata"`
	Role                          string                 `json:"role"`
	AuthenticatorAssuranceLevel   string                 `json:"aal,omitempty"`
	AuthenticationMethodReference []models.AMREntry      `json:"amr,omitempty"`
	SessionId                     string                 `json:"session_id,omitempty"`
}

type MFAVerificationAttemptInput struct {
	UserID   uuid.UUID `json:"user_id"`
	FactorID uuid.UUID `json:"factor_id"`
	Valid    bool      `json:"valid"`
}

type MFAVerificationAttemptOutput struct {
	Decision  string        `json:"decision"`
	Message   string        `json:"message"`
	HookError AuthHookError `json:"error"`
}

type PasswordVerificationAttemptInput struct {
	UserID uuid.UUID `json:"user_id"`
	Valid  bool      `json:"valid"`
}

type PasswordVerificationAttemptOutput struct {
	Decision         string        `json:"decision"`
	Message          string        `json:"message"`
	ShouldLogoutUser bool          `json:"should_logout_user"`
	HookError        AuthHookError `json:"error"`
}

type CustomAccessTokenInput struct {
	UserID uuid.UUID     `json:"user_id"`
	Claims *GoTrueClaims `json:"claims"`
}

type CustomAccessTokenOutput struct {
	Claims    map[string]interface{} `json:"claims"`
	HookError AuthHookError          `json:"error,omitempty"`
}

func (mf *MFAVerificationAttemptOutput) IsError() bool {
	return mf.HookError.Message != ""
}

func (mf *MFAVerificationAttemptOutput) Error() string {
	return mf.HookError.Message
}

func (p *PasswordVerificationAttemptOutput) IsError() bool {
	return p.HookError.Message != ""
}

func (p *PasswordVerificationAttemptOutput) Error() string {
	return p.HookError.Message
}

func (ca *CustomAccessTokenOutput) IsError() bool {
	return ca.HookError.Message != ""
}

func (ca *CustomAccessTokenOutput) Error() string {
	return ca.HookError.Message
}

type AuthHookError struct {
	HTTPCode int    `json:"http_code,omitempty"`
	Message  string `json:"message,omitempty"`
}

func (a *AuthHookError) Error() string {
	return a.Message
}

const (
	DefaultMFAHookRejectionMessage      = "Further MFA verification attempts will be rejected."
	DefaultPasswordHookRejectionMessage = "Further password verification attempts will be rejected."
)
