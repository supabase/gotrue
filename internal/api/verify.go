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
	mail "github.com/supabase/auth/internal/mailer"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/observability"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

const (
	smsVerification         = "sms"
	phoneChangeVerification = "phone_change"
	// includes signupVerification and magicLinkVerification
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
		return badRequestError(ErrorCodeValidationFailed, "Verify requires a verification type")
	}
	switch r.Method {
	case http.MethodGet:
		if p.Token == "" {
			return badRequestError(ErrorCodeValidationFailed, "Verify requires a token or a token hash")
		}
		// TODO: deprecate the token query param from GET /verify and use token_hash instead (breaking change)
		p.TokenHash = p.Token
	case http.MethodPost:
		if (p.Token == "" && p.TokenHash == "") || (p.Token != "" && p.TokenHash != "") {
			return badRequestError(ErrorCodeValidationFailed, "Verify requires either a token or a token hash")
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
					return unprocessableEntityError(ErrorCodeValidationFailed, "Invalid email format").WithInternalError(err)
				}
				p.TokenHash = crypto.GenerateTokenHash(p.Email, p.Token)
			} else {
				return badRequestError(ErrorCodeValidationFailed, "Only an email address or phone number should be provided on verify")
			}
		} else if p.TokenHash != "" {
			if p.Email != "" || p.Phone != "" || p.RedirectTo != "" {
				return badRequestError(ErrorCodeValidationFailed, "Only the token_hash and type should be provided")
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
		// this should have been handled by Chi
		panic("Only GET and POST methods allowed")
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
		rurl        string
	)

	grantParams.FillGrantParams(r)

	flowType := models.ImplicitFlow
	var authenticationMethod models.AuthenticationMethod
	if strings.HasPrefix(params.Token, PKCEPrefix) {
		flowType = models.PKCEFlow
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
		case mail.SignupVerification, mail.InviteVerification:
			user, terr = a.signupVerify(r, ctx, tx, user)
		case mail.RecoveryVerification, mail.MagicLinkVerification:
			user, terr = a.recoverVerify(r, tx, user)
		case mail.EmailChangeVerification:
			user, terr = a.emailChangeVerify(r, tx, params, user)
			if user == nil && terr == nil {
				// only one OTP is confirmed at this point, so we return early and ask the user to confirm the second OTP
				rurl, terr = a.prepRedirectURL(singleConfirmationAccepted, params.RedirectTo, flowType)
				if terr != nil {
					return terr
				}
				return nil
			}
		default:
			return badRequestError(ErrorCodeValidationFailed, "Unsupported verification type")
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
			token, terr = a.issueRefreshToken(r, tx, user, models.OTP, grantParams)
			if terr != nil {
				return terr
			}

			if terr = a.setCookieTokens(config, token, false, w); terr != nil {
				return internalServerError("Failed to set JWT cookie. %s", terr)
			}
		} else if isPKCEFlow(flowType) {
			if authCode, terr = issueAuthCode(tx, user, authenticationMethod); terr != nil {
				return badRequestError(ErrorCodeFlowStateNotFound, "No associated flow state found. %s", terr)
			}
		}
		return nil
	})

	if err != nil {
		var herr *HTTPError
		if errors.As(err, &herr) {
			rurl, err = a.prepErrorRedirectURL(herr, r, params.RedirectTo, flowType)
			if err != nil {
				return err
			}
		}
	}
	if rurl != "" {
		http.Redirect(w, r, rurl, http.StatusSeeOther)
		return nil
	}
	rurl = params.RedirectTo
	if isImplicitFlow(flowType) && token != nil {
		q := url.Values{}
		q.Set("type", params.Type)
		rurl = token.AsRedirectURL(rurl, q)
	} else if isPKCEFlow(flowType) {
		rurl, err = a.prepPKCERedirectURL(rurl, authCode)
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
		case mail.SignupVerification, mail.InviteVerification:
			user, terr = a.signupVerify(r, ctx, tx, user)
		case mail.RecoveryVerification, mail.MagicLinkVerification:
			user, terr = a.recoverVerify(r, tx, user)
		case mail.EmailChangeVerification:
			user, terr = a.emailChangeVerify(r, tx, params, user)
			if user == nil && terr == nil {
				isSingleConfirmationResponse = true
				return nil
			}
		case smsVerification, phoneChangeVerification:
			user, terr = a.smsVerify(r, tx, user, params)
		default:
			return badRequestError(ErrorCodeValidationFailed, "Unsupported verification type")
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
		token, terr = a.issueRefreshToken(r, tx, user, models.OTP, grantParams)
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
			// password generation must succeed
			panic(err)
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

		if user.IsAnonymous {
			user.IsAnonymous = false
			if terr := tx.UpdateOnly(user, "is_anonymous"); terr != nil {
				return terr
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
	log := observability.GetLogEntry(r).Entry
	errorID := utilities.GetRequestID(r.Context())
	err.ErrorID = errorID
	log.WithError(err.Cause()).Info(err.Error())
	if str, ok := oauthErrorMap[err.HTTPStatus]; ok {
		hq.Set("error", str)
		q.Set("error", str)
	}
	hq.Set("error_code", strconv.Itoa(err.HTTPStatus))
	hq.Set("error_description", err.Message)

	q.Set("error_code", strconv.Itoa(err.HTTPStatus))
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
	if flowType == models.PKCEFlow {
		q.Set("message", message)
	}
	u.RawQuery = q.Encode()
	u.Fragment = hq.Encode()
	return u.String(), nil
}

func (a *API) prepPKCERedirectURL(rurl, code string) (string, error) {
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
			currentOTT, terr := models.FindOneTimeToken(tx, params.TokenHash, models.EmailChangeTokenCurrent)
			if terr != nil && !models.IsNotFoundError(terr) {
				return terr
			}

			newOTT, terr := models.FindOneTimeToken(tx, params.TokenHash, models.EmailChangeTokenNew)
			if terr != nil && !models.IsNotFoundError(terr) {
				return terr
			}

			user.EmailChangeConfirmStatus = singleConfirmation

			if currentOTT != nil && params.TokenHash == currentOTT.TokenHash {
				if terr := models.ClearOneTimeTokenForUser(tx, user.ID, models.EmailChangeTokenCurrent); terr != nil {
					return terr
				}
			} else if newOTT != nil && params.TokenHash == newOTT.TokenHash {
				if terr := models.ClearOneTimeTokenForUser(tx, user.ID, models.EmailChangeTokenNew); terr != nil {
					return terr
				}
			}
			if terr := tx.UpdateOnly(user, "email_change_confirm_status"); terr != nil {
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
	case mail.EmailOTPVerification:
		// need to find user by confirmation token or recovery token with the token hash
		user, err = models.FindUserByConfirmationOrRecoveryToken(conn, params.TokenHash)
	case mail.SignupVerification, mail.InviteVerification:
		user, err = models.FindUserByConfirmationToken(conn, params.TokenHash)
	case mail.RecoveryVerification, mail.MagicLinkVerification:
		user, err = models.FindUserByRecoveryToken(conn, params.TokenHash)
	case mail.EmailChangeVerification:
		user, err = models.FindUserByEmailChangeToken(conn, params.TokenHash)
	default:
		return nil, badRequestError(ErrorCodeValidationFailed, "Invalid email verification type")
	}

	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, forbiddenError(ErrorCodeOTPExpired, "Email link is invalid or has expired").WithInternalError(err)
		}
		return nil, internalServerError("Database error finding user from email link").WithInternalError(err)
	}

	if user.IsBanned() {
		return nil, forbiddenError(ErrorCodeUserBanned, "User is banned")
	}

	var isExpired bool
	switch params.Type {
	case mail.EmailOTPVerification:
		sentAt := user.ConfirmationSentAt
		params.Type = "signup"
		if _, err := models.FindOneTimeToken(conn, params.TokenHash, models.RecoveryToken); err != nil {
			if !models.IsNotFoundError(err) {
				return nil, internalServerError("Database error finding token").WithInternalError(err)
			}
			sentAt = user.RecoverySentAt
			params.Type = "magiclink"
		}
		isExpired = isOtpExpired(sentAt, config.Mailer.OtpExp)
	case mail.SignupVerification, mail.InviteVerification:
		isExpired = isOtpExpired(user.ConfirmationSentAt, config.Mailer.OtpExp)
	case mail.RecoveryVerification, mail.MagicLinkVerification:
		isExpired = isOtpExpired(user.RecoverySentAt, config.Mailer.OtpExp)
	case mail.EmailChangeVerification:
		isExpired = isOtpExpired(user.EmailChangeSentAt, config.Mailer.OtpExp)
	}

	if isExpired {
		return nil, forbiddenError(ErrorCodeOTPExpired, "Email link is invalid or has expired").WithInternalMessage("email link has expired")
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
	case mail.EmailChangeVerification:
		// Since the email change could be trigger via the implicit or PKCE flow,
		// the query used has to also check if the token saved in the db contains the pkce_ prefix
		user, err = models.FindUserForEmailChange(conn, params.Email, tokenHash, aud, config.Mailer.SecureEmailChangeEnabled)
	default:
		user, err = models.FindUserByEmailAndAudience(conn, params.Email, aud)
	}

	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, forbiddenError(ErrorCodeOTPExpired, "Token has expired or is invalid").WithInternalError(err)
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	if user.IsBanned() {
		return nil, forbiddenError(ErrorCodeUserBanned, "User is banned")
	}

	var isValid bool
	var otpErr error
	smsProvider, _ := sms_provider.GetSmsProvider(*config)
	switch params.Type {
	case mail.EmailOTPVerification:
		// if the type is emailOTPVerification, we'll check both the confirmation_token and recovery_token columns
		isValid, otpErr = isOtpValid(conn, tokenHash, config.Mailer.OtpExp, models.ConfirmationToken)
		if otpErr == nil {
			params.Type = mail.SignupVerification
			break
		}
		isValid, otpErr = isOtpValid(conn, tokenHash, config.Mailer.OtpExp, models.RecoveryToken)
		if otpErr == nil {
			params.Type = mail.MagicLinkVerification
			break
		}
	case mail.SignupVerification, mail.InviteVerification:
		isValid, otpErr = isOtpValid(conn, tokenHash, config.Mailer.OtpExp, models.ConfirmationToken)
	case mail.RecoveryVerification, mail.MagicLinkVerification:
		isValid, otpErr = isOtpValid(conn, tokenHash, config.Mailer.OtpExp, models.RecoveryToken)
	case mail.EmailChangeVerification:
		isValid, otpErr = isOtpValid(conn, tokenHash, config.Mailer.OtpExp, models.EmailChangeTokenCurrent, models.EmailChangeTokenNew)
	case phoneChangeVerification, smsVerification:
		phone := params.Phone
		if params.Type == phoneChangeVerification {
			phone = user.PhoneChange
		}
		if config.Sms.IsTwilioVerifyProvider() {
			if testOTP, ok := config.Sms.GetTestOTP(phone, time.Now()); ok {
				if params.Token == testOTP {
					return user, nil
				}
			}
			if err := smsProvider.(*sms_provider.TwilioVerifyProvider).VerifyOTP(phone, params.Token); err != nil {
				return nil, forbiddenError(ErrorCodeOTPExpired, "Token has expired or is invalid").WithInternalError(err)
			}
			return user, nil
		}
		isValid, otpErr = isOtpValid(conn, tokenHash, config.Sms.OtpExp, models.ConfirmationToken, models.PhoneChangeToken)
	}

	if otpErr != nil {
		return nil, internalServerError("Database error finding token").WithInternalError(err)
	}

	if !isValid {
		return nil, forbiddenError(ErrorCodeOTPExpired, "Token has expired or is invalid").WithInternalMessage("token has expired or is invalid")
	}
	return user, nil
}

// isOtpValid checks the actual otp sent against the expected otp and ensures that it's within the valid window
func isOtpValid(tx *storage.Connection, tokenHash string, otpExp uint, tokenTypes ...models.OneTimeTokenType) (bool, error) {
	token, err := models.FindOneTimeToken(tx, tokenHash, tokenTypes...)
	if err != nil {
		if !models.IsNotFoundError(err) {
			return false, err
		}
		// try again with the pkce prefix
		token, err = models.FindOneTimeToken(tx, "pkce_"+tokenHash, tokenTypes...)
		if err != nil {
			if models.IsNotFoundError(err) {
				return false, nil
			}
			return false, err
		}
	}
	if isOtpExpired(&token.CreatedAt, otpExp) {
		return false, nil
	}
	return true, nil
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
