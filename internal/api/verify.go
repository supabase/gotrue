package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/sethvargo/go-password/password"
	"github.com/supabase/auth/internal/api/provider"
	"github.com/supabase/auth/internal/api/sms_provider"
	"github.com/supabase/auth/internal/crypto"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/observability"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

const (
	signupVerification      = "signup"
	recoveryVerification    = "recovery"
	inviteVerification      = "invite"
	magicLinkVerification   = "magiclink"
	emailChangeVerification = "email_change"
	smsVerification         = "sms"
	phoneChangeVerification = "phone_change"
	// includes signupVerification and magicLinkVerification
	emailOTPVerification = "email"
)

const (
	zeroConfirmation int = iota
	singleConfirmation
)

// Only applicable when SECURE_EMAIL_CHANGE_ENABLED
const singleConfirmationAccepted = "Confirmation link accepted. Please proceed to confirm link sent to the other email"

// VerifyParams are the parameters the Verify endpoint accepts
type VerifyParams struct {
	Type       string `json:"type"`
	Token      string `json:"token"`
	TokenHash  string `json:"token_hash"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	RedirectTo string `json:"redirect_to"`
}

func (p *VerifyParams) Validate(r *http.Request) error {
	var err error
	if p.Type == "" {
		return badRequestError("Verify requires a verification type")
	}
	switch r.Method {
	case http.MethodGet:
		if p.Token == "" {
			return badRequestError("Verify requires a token or a token hash")
		}
		// TODO: deprecate the token query param from GET /verify and use token_hash instead (breaking change)
		p.TokenHash = p.Token
	case http.MethodPost:
		if (p.Token == "" && p.TokenHash == "") || (p.Token != "" && p.TokenHash != "") {
			return badRequestError("Verify requires either a token or a token hash")
		}
		if p.Token != "" {
			if isPhoneOtpVerification(p) {
				p.Phone, err = validatePhone(p.Phone)
				if err != nil {
					return err
				}
				p.TokenHash = crypto.GenerateTokenHash(p.Phone, p.Token)
			} else if isEmailOtpVerification(p) {
				p.Email, err = validateEmail(p.Email)
				if err != nil {
					return unprocessableEntityError("Invalid email format").WithInternalError(err)
				}
				p.TokenHash = crypto.GenerateTokenHash(p.Email, p.Token)
			} else {
				return badRequestError("Only an email address or phone number should be provided on verify")
			}
		} else if p.TokenHash != "" {
			if p.Email != "" || p.Phone != "" || p.RedirectTo != "" {
				return badRequestError("Only the token_hash and type should be provided")
			}
		}
	default:
		return nil
	}
	return nil
}

// Verify exchanges a confirmation or recovery token to a refresh token
func (a *API) Verify(w http.ResponseWriter, r *http.Request) error {
	params := &VerifyParams{}
	switch r.Method {
	case http.MethodGet:
		params.Token = r.FormValue("token")
		params.Type = r.FormValue("type")
		params.RedirectTo = utilities.GetReferrer(r, a.config)
		if err := params.Validate(r); err != nil {
			return err
		}
		return a.verifyGet(w, r, params)
	case http.MethodPost:
		if err := retrieveRequestParams(r, params); err != nil {
			return err
		}
		if err := params.Validate(r); err != nil {
			return err
		}
		return a.verifyPost(w, r, params)
	default:
		return unprocessableEntityError("Only GET and POST methods are supported.")
	}
}

func (a *API) verifyGet(w http.ResponseWriter, r *http.Request, params *VerifyParams) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config

	var (
		user        *models.User
		grantParams models.GrantParams
		err         error
		token       *AccessTokenResponse
		authCode    string
	)

	grantParams.FillGrantParams(r)

	flowType := models.ImplicitFlow
	var authenticationMethod models.AuthenticationMethod
	// TODO: simplify this
	if strings.HasPrefix(params.Token, PKCEPrefix) {
		flowType = models.PKCEFlow
		authenticationMethod, err = models.ParseAuthenticationMethod(params.Type)
		if err != nil {
			return err
		}
	} else if strings.HasPrefix(params.Token, AuthCodePrefix) {
		flowType = models.AuthCodeFlow
		authenticationMethod, err = models.ParseAuthenticationMethod(params.Type)
		if err != nil {
			return err
		}
	}
	err = db.Transaction(func(tx *storage.Connection) error {
		var terr error
		user, terr = a.verifyTokenHash(tx, params)
		if terr != nil {
			return terr
		}
		switch params.Type {
		case signupVerification, inviteVerification:
			user, terr = a.signupVerify(r, ctx, tx, user)
		case recoveryVerification, magicLinkVerification:
			user, terr = a.recoverVerify(r, tx, user)
		case emailChangeVerification:
			user, terr = a.emailChangeVerify(r, tx, params, user)
			if user == nil && terr == nil {
				// when double confirmation is required
				rurl, err := a.prepRedirectURL(singleConfirmationAccepted, params.RedirectTo, flowType)
				if err != nil {
					return err
				}
				http.Redirect(w, r, rurl, http.StatusSeeOther)
				return nil
			}
		default:
			return unprocessableEntityError("Unsupported verification type")
		}

		if terr != nil {
			return terr
		}

		if terr := user.UpdateAppMetaDataProviders(tx); terr != nil {
			return terr
		}

		// Reload user model from db.
		// This is important for refreshing the data in any generated columns like IsAnonymous.
		if terr := tx.Reload(user); err != nil {
			return terr
		}

		if isImplicitFlow(flowType) {
			token, terr = a.issueRefreshToken(ctx, tx, user, models.OTP, grantParams)
			if terr != nil {
				return terr
			}

			if terr = a.setCookieTokens(config, token, false, w); terr != nil {
				return internalServerError("Failed to set JWT cookie. %s", terr)
			}
		} else if isCodeFlow(flowType) {
			if authCode, terr = issueAuthCode(tx, user, authenticationMethod); terr != nil {
				return badRequestError("No associated flow state found. %s", terr)
			}
		}
		return nil
	})

	if err != nil {
		var herr *HTTPError
		if errors.As(err, &herr) {
			rurl, err := a.prepErrorRedirectURL(herr, r, params.RedirectTo, flowType)
			if err != nil {
				return err
			}
			http.Redirect(w, r, rurl, http.StatusSeeOther)
			return nil
		}
	}
	rurl := params.RedirectTo
	if isImplicitFlow(flowType) && token != nil {
		q := url.Values{}
		q.Set("type", params.Type)
		rurl = token.AsRedirectURL(rurl, q)
	} else if isCodeFlow(flowType) {
		rurl, err = a.prepCodeRedirectURL(rurl, authCode)
		if err != nil {
			return err
		}
	}
	http.Redirect(w, r, rurl, http.StatusSeeOther)
	return nil
}

func (a *API) verifyPost(w http.ResponseWriter, r *http.Request, params *VerifyParams) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config

	var (
		user        *models.User
		grantParams models.GrantParams
		token       *AccessTokenResponse
	)
	var isSingleConfirmationResponse = false

	grantParams.FillGrantParams(r)

	err := db.Transaction(func(tx *storage.Connection) error {
		var terr error
		aud := a.requestAud(ctx, r)

		if isUsingTokenHash(params) {
			user, terr = a.verifyTokenHash(tx, params)
		} else {
			user, terr = a.verifyUserAndToken(tx, params, aud)
		}
		if terr != nil {
			return terr
		}

		switch params.Type {
		case signupVerification, inviteVerification:
			user, terr = a.signupVerify(r, ctx, tx, user)
		case recoveryVerification, magicLinkVerification:
			user, terr = a.recoverVerify(r, tx, user)
		case emailChangeVerification:
			user, terr = a.emailChangeVerify(r, tx, params, user)
			if user == nil && terr == nil {
				isSingleConfirmationResponse = true
				return nil
			}
		case smsVerification, phoneChangeVerification:
			user, terr = a.smsVerify(r, tx, user, params)
		default:
			return unprocessableEntityError("Unsupported verification type")
		}

		if terr != nil {
			return terr
		}

		if terr := user.UpdateAppMetaDataProviders(tx); terr != nil {
			return terr
		}

		// Reload user model from db.
		// This is important for refreshing the data in any generated columns like IsAnonymous.
		if terr := tx.Reload(user); terr != nil {
			return terr
		}
		token, terr = a.issueRefreshToken(ctx, tx, user, models.OTP, grantParams)
		if terr != nil {
			return terr
		}

		if terr = a.setCookieTokens(config, token, false, w); terr != nil {
			return internalServerError("Failed to set JWT cookie. %s", terr)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if isSingleConfirmationResponse {
		return sendJSON(w, http.StatusOK, map[string]string{
			"msg":  singleConfirmationAccepted,
			"code": strconv.Itoa(http.StatusOK),
		})
	}
	return sendJSON(w, http.StatusOK, token)
}

func (a *API) signupVerify(r *http.Request, ctx context.Context, conn *storage.Connection, user *models.User) (*models.User, error) {
	if user.EncryptedPassword == "" && user.InvitedAt != nil {
		// sign them up with temporary password, and require application
		// to present the user with a password set form
		password, err := password.Generate(64, 10, 0, false, true)
		if err != nil {
			return nil, err
		}

		if err := user.SetPassword(ctx, password); err != nil {
			return nil, err
		}
	}

	err := conn.Transaction(func(tx *storage.Connection) error {
		var terr error
		if user.EncryptedPassword == "" && user.InvitedAt != nil {
			if terr = user.UpdatePassword(tx, nil); terr != nil {
				return internalServerError("Error storing password").WithInternalError(terr)
			}
		}

		if terr = models.NewAuditLogEntry(r, tx, user, models.UserSignedUpAction, "", nil); terr != nil {
			return terr
		}

		if terr = user.Confirm(tx); terr != nil {
			return internalServerError("Error confirming user").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a *API) recoverVerify(r *http.Request, conn *storage.Connection, user *models.User) (*models.User, error) {
	err := conn.Transaction(func(tx *storage.Connection) error {
		var terr error
		if terr = user.Recover(tx); terr != nil {
			return terr
		}
		if !user.IsConfirmed() {
			if terr = models.NewAuditLogEntry(r, tx, user, models.UserSignedUpAction, "", nil); terr != nil {
				return terr
			}

			if terr = user.Confirm(tx); terr != nil {
				return terr
			}
		} else {
			if terr = models.NewAuditLogEntry(r, tx, user, models.LoginAction, "", nil); terr != nil {
				return terr
			}
		}
		return nil
	})

	if err != nil {
		return nil, internalServerError("Database error updating user").WithInternalError(err)
	}
	return user, nil
}

func (a *API) smsVerify(r *http.Request, conn *storage.Connection, user *models.User, params *VerifyParams) (*models.User, error) {

	err := conn.Transaction(func(tx *storage.Connection) error {

		if params.Type == smsVerification {
			if terr := models.NewAuditLogEntry(r, tx, user, models.UserSignedUpAction, "", nil); terr != nil {
				return terr
			}
			if terr := user.ConfirmPhone(tx); terr != nil {
				return internalServerError("Error confirming user").WithInternalError(terr)
			}
		} else if params.Type == phoneChangeVerification {
			if terr := models.NewAuditLogEntry(r, tx, user, models.UserModifiedAction, "", nil); terr != nil {
				return terr
			}
			if identity, terr := models.FindIdentityByIdAndProvider(tx, user.ID.String(), "phone"); terr != nil {
				if !models.IsNotFoundError(terr) {
					return terr
				}
				// confirming the phone change should create a new phone identity if the user doesn't have one
				if _, terr = a.createNewIdentity(tx, user, "phone", structs.Map(provider.Claims{
					Subject:       user.ID.String(),
					Phone:         params.Phone,
					PhoneVerified: true,
				})); terr != nil {
					return terr
				}
			} else {
				if terr := identity.UpdateIdentityData(tx, map[string]interface{}{
					"phone":          params.Phone,
					"phone_verified": true,
				}); terr != nil {
					return terr
				}
			}
			if terr := user.ConfirmPhoneChange(tx); terr != nil {
				return internalServerError("Error confirming user").WithInternalError(terr)
			}
		}

		if terr := tx.Load(user, "Identities"); terr != nil {
			return internalServerError("Error refetching identities").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a *API) prepErrorRedirectURL(err *HTTPError, r *http.Request, rurl string, flowType models.FlowType) (string, error) {
	u, perr := url.Parse(rurl)
	if perr != nil {
		return "", err
	}
	q := u.Query()

	// Maintain separate query params for hash and query
	hq := url.Values{}
	log := observability.GetLogEntry(r)
	errorID := getRequestID(r.Context())
	err.ErrorID = errorID
	log.WithError(err.Cause()).Info(err.Error())
	if str, ok := oauthErrorMap[err.Code]; ok {
		hq.Set("error", str)
		q.Set("error", str)
	}
	hq.Set("error_code", strconv.Itoa(err.Code))
	hq.Set("error_description", err.Message)

	q.Set("error_code", strconv.Itoa(err.Code))
	q.Set("error_description", err.Message)
	if flowType == models.PKCEFlow {
		// Additionally, may override existing error query param if set to PKCE.
		u.RawQuery = q.Encode()
	}
	// Left as hash fragment to comply with spec.
	u.Fragment = hq.Encode()
	return u.String(), nil
}

func (a *API) prepRedirectURL(message string, rurl string, flowType models.FlowType) (string, error) {
	u, perr := url.Parse(rurl)
	if perr != nil {
		return "", perr
	}
	hq := url.Values{}
	q := u.Query()
	hq.Set("message", message)
	if flowType == models.PKCEFlow || flowType == models.AuthCodeFlow {
		q.Set("message", message)
	}
	u.RawQuery = q.Encode()
	u.Fragment = hq.Encode()
	return u.String(), nil
}

func (a *API) prepCodeRedirectURL(rurl, code string) (string, error) {
	u, err := url.Parse(rurl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("code", code)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (a *API) emailChangeVerify(r *http.Request, conn *storage.Connection, params *VerifyParams, user *models.User) (*models.User, error) {
	config := a.config
	if config.Mailer.SecureEmailChangeEnabled && user.EmailChangeConfirmStatus == zeroConfirmation && user.GetEmail() != "" {
		err := conn.Transaction(func(tx *storage.Connection) error {
			user.EmailChangeConfirmStatus = singleConfirmation
			if params.Token == user.EmailChangeTokenCurrent || params.TokenHash == user.EmailChangeTokenCurrent {
				user.EmailChangeTokenCurrent = ""
			} else if params.Token == user.EmailChangeTokenNew || params.TokenHash == user.EmailChangeTokenNew {
				user.EmailChangeTokenNew = ""
			}
			if terr := tx.UpdateOnly(user, "email_change_confirm_status", "email_change_token_current", "email_change_token_new"); terr != nil {
				return terr
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	// one email is confirmed at this point if GOTRUE_MAILER_SECURE_EMAIL_CHANGE_ENABLED is enabled
	err := conn.Transaction(func(tx *storage.Connection) error {
		if terr := models.NewAuditLogEntry(r, tx, user, models.UserModifiedAction, "", nil); terr != nil {
			return terr
		}

		if identity, terr := models.FindIdentityByIdAndProvider(tx, user.ID.String(), "email"); terr != nil {
			if !models.IsNotFoundError(terr) {
				return terr
			}
			// confirming the email change should create a new email identity if the user doesn't have one
			if _, terr = a.createNewIdentity(tx, user, "email", structs.Map(provider.Claims{
				Subject:       user.ID.String(),
				Email:         user.EmailChange,
				EmailVerified: true,
			})); terr != nil {
				return terr
			}
		} else {
			if terr := identity.UpdateIdentityData(tx, map[string]interface{}{
				"email":          user.EmailChange,
				"email_verified": true,
			}); terr != nil {
				return terr
			}
		}
		if user.IsAnonymous {
			user.IsAnonymous = false
			if terr := tx.UpdateOnly(user, "is_anonymous"); terr != nil {
				return terr
			}
		}
		if terr := tx.Load(user, "Identities"); terr != nil {
			return internalServerError("Error refetching identities").WithInternalError(terr)
		}
		if terr := user.ConfirmEmailChange(tx, zeroConfirmation); terr != nil {
			return internalServerError("Error confirm email").WithInternalError(terr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (a *API) verifyTokenHash(conn *storage.Connection, params *VerifyParams) (*models.User, error) {
	config := a.config

	var user *models.User
	var err error
	switch params.Type {
	case emailOTPVerification:
		// need to find user by confirmation token or recovery token with the token hash
		user, err = models.FindUserByConfirmationOrRecoveryToken(conn, params.TokenHash)
	case signupVerification, inviteVerification:
		user, err = models.FindUserByConfirmationToken(conn, params.TokenHash)
	case recoveryVerification, magicLinkVerification:
		user, err = models.FindUserByRecoveryToken(conn, params.TokenHash)
	case emailChangeVerification:
		user, err = models.FindUserByEmailChangeToken(conn, params.TokenHash)
	default:
		return nil, badRequestError("Invalid email verification type")
	}

	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, expiredTokenError("Email link is invalid or has expired").WithInternalError(err)
		}
		return nil, internalServerError("Database error finding user from email link").WithInternalError(err)
	}

	if user.IsBanned() {
		return nil, unauthorizedError("Error confirming user").WithInternalMessage("user is banned")
	}

	var isExpired bool
	switch params.Type {
	case emailOTPVerification:
		sentAt := user.ConfirmationSentAt
		params.Type = "signup"
		if user.RecoveryToken == params.TokenHash {
			sentAt = user.RecoverySentAt
			params.Type = "magiclink"
		}
		isExpired = isOtpExpired(sentAt, config.Mailer.OtpExp)
	case signupVerification, inviteVerification:
		isExpired = isOtpExpired(user.ConfirmationSentAt, config.Mailer.OtpExp)
	case recoveryVerification, magicLinkVerification:
		isExpired = isOtpExpired(user.RecoverySentAt, config.Mailer.OtpExp)
	case emailChangeVerification:
		isExpired = isOtpExpired(user.EmailChangeSentAt, config.Mailer.OtpExp)
	}

	if isExpired {
		return nil, expiredTokenError("Email link is invalid or has expired").WithInternalMessage("email link has expired")
	}

	return user, nil
}

// verifyUserAndToken verifies the token associated to the user based on the verify type
func (a *API) verifyUserAndToken(conn *storage.Connection, params *VerifyParams, aud string) (*models.User, error) {
	config := a.config

	var user *models.User
	var err error
	tokenHash := params.TokenHash

	switch params.Type {
	case phoneChangeVerification:
		user, err = models.FindUserByPhoneChangeAndAudience(conn, params.Phone, aud)
	case smsVerification:
		user, err = models.FindUserByPhoneAndAudience(conn, params.Phone, aud)
	case emailChangeVerification:
		// Since the email change could be trigger via the implicit or PKCE flow,
		// the query used has to also check if the token saved in the db contains the pkce_ prefix
		user, err = models.FindUserForEmailChange(conn, params.Email, tokenHash, aud, config.Mailer.SecureEmailChangeEnabled)
	default:
		user, err = models.FindUserByEmailAndAudience(conn, params.Email, aud)
	}

	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, expiredTokenError("Token has expired or is invalid").WithInternalError(err)
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	if user.IsBanned() {
		return nil, unauthorizedError("Error confirming user").WithInternalMessage("user is banned")
	}

	var isValid bool

	smsProvider, _ := sms_provider.GetSmsProvider(*config)
	switch params.Type {
	case emailOTPVerification:
		// if the type is emailOTPVerification, we'll check both the confirmation_token and recovery_token columns
		if isOtpValid(tokenHash, user.ConfirmationToken, user.ConfirmationSentAt, config.Mailer.OtpExp) {
			isValid = true
			params.Type = signupVerification
		} else if isOtpValid(tokenHash, user.RecoveryToken, user.RecoverySentAt, config.Mailer.OtpExp) {
			isValid = true
			params.Type = magicLinkVerification
		} else {
			isValid = false
		}
	case signupVerification, inviteVerification:
		isValid = isOtpValid(tokenHash, user.ConfirmationToken, user.ConfirmationSentAt, config.Mailer.OtpExp)
	case recoveryVerification, magicLinkVerification:
		isValid = isOtpValid(tokenHash, user.RecoveryToken, user.RecoverySentAt, config.Mailer.OtpExp)
	case emailChangeVerification:
		isValid = isOtpValid(tokenHash, user.EmailChangeTokenCurrent, user.EmailChangeSentAt, config.Mailer.OtpExp) ||
			isOtpValid(tokenHash, user.EmailChangeTokenNew, user.EmailChangeSentAt, config.Mailer.OtpExp)
	case phoneChangeVerification, smsVerification:
		phone := params.Phone
		sentAt := user.ConfirmationSentAt
		expectedToken := user.ConfirmationToken
		if params.Type == phoneChangeVerification {
			phone = user.PhoneChange
			sentAt = user.PhoneChangeSentAt
			expectedToken = user.PhoneChangeToken
		}
		if config.Sms.IsTwilioVerifyProvider() {
			if testOTP, ok := config.Sms.GetTestOTP(params.Phone, time.Now()); ok {
				if params.Token == testOTP {
					return user, nil
				}
			}
			if err := smsProvider.(*sms_provider.TwilioVerifyProvider).VerifyOTP(phone, params.Token); err != nil {
				return nil, expiredTokenError("Token has expired or is invalid").WithInternalError(err)
			}
			return user, nil
		}
		isValid = isOtpValid(tokenHash, expectedToken, sentAt, config.Sms.OtpExp)
	}

	if !isValid {
		return nil, expiredTokenError("Token has expired or is invalid").WithInternalMessage("token has expired or is invalid")
	}
	return user, nil
}

// isOtpValid checks the actual otp sent against the expected otp and ensures that it's within the valid window
func isOtpValid(actual, expected string, sentAt *time.Time, otpExp uint) bool {
	if expected == "" || sentAt == nil {
		return false
	}
	return !isOtpExpired(sentAt, otpExp) && ((actual == expected) || ("pkce_"+actual == expected))
}

func isOtpExpired(sentAt *time.Time, otpExp uint) bool {
	return time.Now().After(sentAt.Add(time.Second * time.Duration(otpExp)))
}

// isPhoneOtpVerification checks if the verification came from a phone otp
func isPhoneOtpVerification(params *VerifyParams) bool {
	return params.Phone != "" && params.Email == ""
}

// isEmailOtpVerification checks if the verification came from an email otp
func isEmailOtpVerification(params *VerifyParams) bool {
	return params.Phone == "" && params.Email != ""
}

func isUsingTokenHash(params *VerifyParams) bool {
	return params.TokenHash != "" && params.Token == "" && params.Phone == "" && params.Email == ""
}
