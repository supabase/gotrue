package hooks

import (
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt"
	"github.com/supabase/auth/internal/models"
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

type HTTPHookInput interface {
	IsHTTPHook()
}

type HookOutput interface {
	IsError() bool
	Error() string
}

// #nosec
const MinimumViableTokenSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "aud": {
      "type": "string"
    },
    "exp": {
      "type": "integer"
    },
    "jti": {
      "type": "string"
    },
    "iat": {
      "type": "integer"
    },
    "iss": {
      "type": "string"
    },
    "nbf": {
      "type": "integer"
    },
    "sub": {
      "type": "string"
    },
    "email": {
      "type": "string"
    },
    "phone": {
      "type": "string"
    },
    "app_metadata": {
      "type": "object",
      "additionalProperties": true
    },
    "user_metadata": {
      "type": "object",
      "additionalProperties": true
    },
    "role": {
      "type": "string"
    },
    "aal": {
      "type": "string"
    },
    "amr": {
      "type": "array",
      "items": {
        "type": "object"
      }
    },
    "session_id": {
      "type": "string"
    }
  },
  "required": ["aud", "exp", "iat", "sub", "email", "phone", "role", "aal", "session_id"]
}`

// AccessTokenClaims is a struct thats used for JWT claims
type AccessTokenClaims struct {
	jwt.StandardClaims
	Email                         string                 `json:"email"`
	Phone                         string                 `json:"phone"`
	AppMetaData                   map[string]interface{} `json:"app_metadata"`
	UserMetaData                  map[string]interface{} `json:"user_metadata"`
	Role                          string                 `json:"role"`
	AuthenticatorAssuranceLevel   string                 `json:"aal,omitempty"`
	AuthenticationMethodReference []models.AMREntry      `json:"amr,omitempty"`
	SessionId                     string                 `json:"session_id,omitempty"`
	IsAnonymous                   bool                   `json:"is_anonymous"`
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
	UserID               uuid.UUID          `json:"user_id"`
	Claims               *AccessTokenClaims `json:"claims"`
	AuthenticationMethod string             `json:"authentication_method"`
}

type CustomAccessTokenOutput struct {
	Claims    map[string]interface{} `json:"claims"`
	HookError AuthHookError          `json:"error,omitempty"`
}

type CustomSMSProviderInput struct {
	UserID uuid.UUID `json:"user_id"`
	Phone  string    `json:"phone"`
	OTP    string    `json:"otp"`
}

type CustomSMSProviderOutput struct {
	Success   bool          `json:"success"`
	HookError AuthHookError `json:"error,omitempty"`
}

type CustomEmailProviderInput struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	OTP    string    `json:"otp"`
}

type CustomEmailProviderOutput struct {
	Success   bool          `json:"success"`
	HookError AuthHookError `json:"error,omitempty"`
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

func (cs *CustomSMSProviderOutput) IsError() bool {
	return cs.HookError.Message != ""
}

func (cs *CustomSMSProviderOutput) Error() string {
	return cs.HookError.Message
}

func (cs *CustomSMSProviderOutput) IsHTTPHook() bool {
	return true
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
