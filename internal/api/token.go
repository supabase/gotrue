package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"fmt"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/xeipuuv/gojsonschema"

	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/hooks"
	"github.com/supabase/auth/internal/metering"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/observability"
	"github.com/supabase/auth/internal/storage"
)

// AccessTokenClaims is a struct thats used for JWT claims
type AccessTokenClaims struct {
	jwt.RegisteredClaims
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

// AccessTokenResponse represents an OAuth2 success response
type AccessTokenResponse struct {
	Token                string             `json:"access_token"`
	TokenType            string             `json:"token_type"` // Bearer
	ExpiresIn            int                `json:"expires_in"`
	ExpiresAt            int64              `json:"expires_at"`
	RefreshToken         string             `json:"refresh_token"`
	User                 *models.User       `json:"user"`
	ProviderAccessToken  string             `json:"provider_token,omitempty"`
	ProviderRefreshToken string             `json:"provider_refresh_token,omitempty"`
	WeakPassword         *WeakPasswordError `json:"weak_password,omitempty"`
}

// AsRedirectURL encodes the AccessTokenResponse as a redirect URL that
// includes the access token response data in a URL fragment.
func (r *AccessTokenResponse) AsRedirectURL(redirectURL string, extraParams url.Values) string {
	extraParams.Set("access_token", r.Token)
	extraParams.Set("token_type", r.TokenType)
	extraParams.Set("expires_in", strconv.Itoa(r.ExpiresIn))
	extraParams.Set("expires_at", strconv.FormatInt(r.ExpiresAt, 10))
	extraParams.Set("refresh_token", r.RefreshToken)

	return redirectURL + "#" + extraParams.Encode()
}

// PasswordGrantParams are the parameters the ResourceOwnerPasswordGrant method accepts
type PasswordGrantParams struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// PKCEGrantParams are the parameters the PKCEGrant method accepts
type PKCEGrantParams struct {
	AuthCode     string `json:"auth_code"`
	CodeVerifier string `json:"code_verifier"`
}

const useCookieHeader = "x-use-cookie"
const InvalidLoginMessage = "Invalid login credentials"

// Token is the endpoint for OAuth access token requests
func (a *API) Token(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	grantType := r.FormValue("grant_type")
	switch grantType {
	case "password":
		return a.ResourceOwnerPasswordGrant(ctx, w, r)
	case "refresh_token":
		return a.RefreshTokenGrant(ctx, w, r)
	case "id_token":
		return a.IdTokenGrant(ctx, w, r)
	case "pkce":
		return a.PKCE(ctx, w, r)
	default:
		return oauthError("unsupported_grant_type", "")
	}
}

// ResourceOwnerPasswordGrant implements the password grant type flow
func (a *API) ResourceOwnerPasswordGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	db := a.db.WithContext(ctx)

	params := &PasswordGrantParams{}
	if err := retrieveRequestParams(r, params); err != nil {
		return err
	}

	aud := a.requestAud(ctx, r)
	config := a.config

	if params.Email != "" && params.Phone != "" {
		return badRequestError(ErrorCodeValidationFailed, "Only an email address or phone number should be provided on login.")
	}
	var user *models.User
	var grantParams models.GrantParams
	var provider string
	var err error

	grantParams.FillGrantParams(r)

	if params.Email != "" {
		provider = "email"
		if !config.External.Email.Enabled {
			return unprocessableEntityError(ErrorCodeEmailProviderDisabled, "Email logins are disabled")
		}
		user, err = models.FindUserByEmailAndAudience(db, params.Email, aud)
	} else if params.Phone != "" {
		provider = "phone"
		if !config.External.Phone.Enabled {
			return unprocessableEntityError(ErrorCodePhoneProviderDisabled, "Phone logins are disabled")
		}
		params.Phone = formatPhoneNumber(params.Phone)
		user, err = models.FindUserByPhoneAndAudience(db, params.Phone, aud)
	} else {
		return oauthError("invalid_grant", InvalidLoginMessage)
	}

	if err != nil {
		if models.IsNotFoundError(err) {
			return oauthError("invalid_grant", InvalidLoginMessage)
		}
		return internalServerError("Database error querying schema").WithInternalError(err)
	}

	if user.IsBanned() {
		return oauthError("invalid_grant", InvalidLoginMessage)
	}

	isValidPassword, shouldReEncrypt, err := user.Authenticate(ctx, db, params.Password, config.Security.DBEncryption.DecryptionKeys, config.Security.DBEncryption.Encrypt, config.Security.DBEncryption.EncryptionKeyID)
	if err != nil {
		return err
	}

	var weakPasswordError *WeakPasswordError
	if isValidPassword {
		if err := a.checkPasswordStrength(ctx, params.Password); err != nil {
			if wpe, ok := err.(*WeakPasswordError); ok {
				weakPasswordError = wpe
			} else {
				observability.GetLogEntry(r).Entry.WithError(err).Warn("Password strength check on sign-in failed")
			}
		}

		if shouldReEncrypt {
			if err := user.SetPassword(ctx, params.Password, true, config.Security.DBEncryption.EncryptionKeyID, config.Security.DBEncryption.EncryptionKey); err != nil {
				return err
			}

			// directly change this in the database without
			// calling user.UpdatePassword() because this
			// is not a password change, just encryption
			// change in the database
			if err := db.UpdateOnly(user, "encrypted_password"); err != nil {
				return err
			}
		}
	}

	if config.Hook.PasswordVerificationAttempt.Enabled {
		input := hooks.PasswordVerificationAttemptInput{
			UserID: user.ID,
			Valid:  isValidPassword,
		}
		output := hooks.PasswordVerificationAttemptOutput{}
		err := a.invokeHook(nil, r, &input, &output, a.config.Hook.PasswordVerificationAttempt.URI)
		if err != nil {
			return err
		}

		if output.Decision == hooks.HookRejection {
			if output.Message == "" {
				output.Message = hooks.DefaultPasswordHookRejectionMessage
			}
			if output.ShouldLogoutUser {
				if err := models.Logout(a.db, user.ID); err != nil {
					return err
				}
			}
			return oauthError("invalid_grant", InvalidLoginMessage)
		}
	}
	if !isValidPassword {
		return oauthError("invalid_grant", InvalidLoginMessage)
	}

	if params.Email != "" && !user.IsConfirmed() {
		return oauthError("invalid_grant", "Email not confirmed")
	} else if params.Phone != "" && !user.IsPhoneConfirmed() {
		return oauthError("invalid_grant", "Phone not confirmed")
	}

	var token *AccessTokenResponse
	err = db.Transaction(func(tx *storage.Connection) error {
		var terr error
		if terr = models.NewAuditLogEntry(r, tx, user, models.LoginAction, "", map[string]interface{}{
			"provider": provider,
		}); terr != nil {
			return terr
		}
		token, terr = a.issueRefreshToken(r, tx, user, models.PasswordGrant, grantParams)
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

	token.WeakPassword = weakPasswordError

	metering.RecordLogin("password", user.ID)
	return sendJSON(w, http.StatusOK, token)
}

func (a *API) PKCE(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	db := a.db.WithContext(ctx)
	var grantParams models.GrantParams

	// There is a slight problem with this as it will pick-up the
	// User-Agent and IP addresses from the server if used on the server
	// side. Currently there's no mechanism to distinguish, but the server
	// can be told to at least propagate the User-Agent header.
	grantParams.FillGrantParams(r)

	params := &PKCEGrantParams{}
	if err := retrieveRequestParams(r, params); err != nil {
		return err
	}

	if params.AuthCode == "" || params.CodeVerifier == "" {
		return badRequestError(ErrorCodeValidationFailed, "invalid request: both auth code and code verifier should be non-empty")
	}

	flowState, err := models.FindFlowStateByAuthCode(db, params.AuthCode)
	// Sanity check in case user ID was not set properly
	if models.IsNotFoundError(err) || flowState.UserID == nil {
		return notFoundError(ErrorCodeFlowStateNotFound, "invalid flow state, no valid flow state found")
	} else if err != nil {
		return err
	}
	if flowState.IsExpired(a.config.External.FlowStateExpiryDuration) {
		return unprocessableEntityError(ErrorCodeFlowStateExpired, "invalid flow state, flow state has expired")
	}

	user, err := models.FindUserByID(db, *flowState.UserID)
	if err != nil {
		return err
	}
	if err := flowState.VerifyPKCE(params.CodeVerifier); err != nil {
		return badRequestError(ErrorBadCodeVerifier, err.Error())
	}

	var token *AccessTokenResponse
	err = db.Transaction(func(tx *storage.Connection) error {
		var terr error
		authMethod, err := models.ParseAuthenticationMethod(flowState.AuthenticationMethod)
		if err != nil {
			return err
		}
		if terr := models.NewAuditLogEntry(r, tx, user, models.LoginAction, "", map[string]interface{}{
			"provider_type": flowState.ProviderType,
		}); terr != nil {
			return terr
		}
		token, terr = a.issueRefreshToken(r, tx, user, authMethod, grantParams)
		if terr != nil {
			return oauthError("server_error", terr.Error())
		}
		token.ProviderAccessToken = flowState.ProviderAccessToken
		// Because not all providers give out a refresh token
		// See corresponding OAuth2 spec: <https://www.rfc-editor.org/rfc/rfc6749.html#section-5.1>
		if flowState.ProviderRefreshToken != "" {
			token.ProviderRefreshToken = flowState.ProviderRefreshToken
		}
		if terr = tx.Destroy(flowState); terr != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, token)
}

func (a *API) generateAccessToken(r *http.Request, tx *storage.Connection, user *models.User, sessionId *uuid.UUID, authenticationMethod models.AuthenticationMethod) (string, int64, error) {
	config := a.config
	if sessionId == nil {
		return "", 0, internalServerError("Session is required to issue access token")
	}
	sid := sessionId.String()
	session, terr := models.FindSessionByID(tx, *sessionId, false)
	if terr != nil {
		return "", 0, terr
	}
	aal, amr, terr := session.CalculateAALAndAMR(user)
	if terr != nil {
		return "", 0, terr
	}

	issuedAt := time.Now().UTC()
	expiresAt := issuedAt.Add(time.Second * time.Duration(config.JWT.Exp))

	claims := &hooks.AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Audience:  []string{user.Aud},
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    config.JWT.Issuer,
		},
		Email:                         user.GetEmail(),
		Phone:                         user.GetPhone(),
		AppMetaData:                   user.AppMetaData,
		UserMetaData:                  user.UserMetaData,
		Role:                          user.Role,
		SessionId:                     sid,
		AuthenticatorAssuranceLevel:   aal.String(),
		AuthenticationMethodReference: amr,
		IsAnonymous:                   user.IsAnonymous,
	}

	var token *jwt.Token
	if config.Hook.CustomAccessToken.Enabled {
		input := hooks.CustomAccessTokenInput{
			UserID:               user.ID,
			Claims:               claims,
			AuthenticationMethod: authenticationMethod.String(),
		}

		output := hooks.CustomAccessTokenOutput{}

		err := a.invokeHook(tx, r, &input, &output, a.config.Hook.CustomAccessToken.URI)
		if err != nil {
			return "", 0, err
		}
		goTrueClaims := jwt.MapClaims(output.Claims)

		token = jwt.NewWithClaims(jwt.SigningMethodHS256, goTrueClaims)

	} else {
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	}

	if config.JWT.KeyID != "" {
		if token.Header == nil {
			token.Header = make(map[string]interface{})
		}

		token.Header["kid"] = config.JWT.KeyID
	}

	signed, err := token.SignedString([]byte(config.JWT.Secret))
	if err != nil {
		return "", 0, err
	}

	return signed, expiresAt.Unix(), nil
}

func (a *API) issueRefreshToken(r *http.Request, conn *storage.Connection, user *models.User, authenticationMethod models.AuthenticationMethod, grantParams models.GrantParams) (*AccessTokenResponse, error) {
	config := a.config

	now := time.Now()
	user.LastSignInAt = &now

	var tokenString string
	var expiresAt int64
	var refreshToken *models.RefreshToken

	err := conn.Transaction(func(tx *storage.Connection) error {
		var terr error

		refreshToken, terr = models.GrantAuthenticatedUser(tx, user, grantParams)
		if terr != nil {
			return internalServerError("Database error granting user").WithInternalError(terr)
		}

		terr = models.AddClaimToSession(tx, *refreshToken.SessionId, authenticationMethod)
		if terr != nil {
			return terr
		}

		tokenString, expiresAt, terr = a.generateAccessToken(r, tx, user, refreshToken.SessionId, authenticationMethod)
		if terr != nil {
			// Account for Hook Error
			httpErr, ok := terr.(*HTTPError)
			if ok {
				return httpErr
			}
			return internalServerError("error generating jwt token").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    config.JWT.Exp,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken.Token,
		User:         user,
	}, nil
}

func (a *API) updateMFASessionAndClaims(r *http.Request, tx *storage.Connection, user *models.User, authenticationMethod models.AuthenticationMethod, grantParams models.GrantParams) (*AccessTokenResponse, error) {
	ctx := r.Context()
	config := a.config
	var tokenString string
	var expiresAt int64
	var refreshToken *models.RefreshToken
	currentClaims := getClaims(ctx)
	sessionId, err := uuid.FromString(currentClaims.SessionId)
	if err != nil {
		return nil, internalServerError("Cannot read SessionId claim as UUID").WithInternalError(err)
	}

	err = tx.Transaction(func(tx *storage.Connection) error {
		if terr := models.AddClaimToSession(tx, sessionId, authenticationMethod); terr != nil {
			return terr
		}
		session, terr := models.FindSessionByID(tx, sessionId, false)
		if terr != nil {
			return terr
		}
		currentToken, terr := models.FindTokenBySessionID(tx, &session.ID)
		if terr != nil {
			return terr
		}
		if err := tx.Load(user, "Identities"); err != nil {
			return err
		}
		// Swap to ensure current token is the latest one
		refreshToken, terr = models.GrantRefreshTokenSwap(r, tx, user, currentToken)
		if terr != nil {
			return terr
		}
		aal, _, terr := session.CalculateAALAndAMR(user)
		if terr != nil {
			return terr
		}

		if err := session.UpdateAALAndAssociatedFactor(tx, aal, grantParams.FactorID); err != nil {
			return err
		}

		tokenString, expiresAt, terr = a.generateAccessToken(r, tx, user, &session.ID, models.TOTPSignIn)
		if terr != nil {
			httpErr, ok := terr.(*HTTPError)
			if ok {
				return httpErr
			}
			return internalServerError("error generating jwt token").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    config.JWT.Exp,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken.Token,
		User:         user,
	}, nil
}

// setCookieTokens sets the access_token & refresh_token in the cookies
func (a *API) setCookieTokens(config *conf.GlobalConfiguration, token *AccessTokenResponse, session bool, w http.ResponseWriter) error {
	// don't need to catch error here since we always set the cookie name
	_ = a.setCookieToken(config, "access-token", token.Token, session, w)
	_ = a.setCookieToken(config, "refresh-token", token.RefreshToken, session, w)
	return nil
}

func (a *API) setCookieToken(config *conf.GlobalConfiguration, name string, tokenString string, session bool, w http.ResponseWriter) error {
	if name == "" {
		return errors.New("failed to set cookie, invalid name")
	}
	cookieName := config.Cookie.Key + "-" + name
	exp := time.Second * time.Duration(config.Cookie.Duration)
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    tokenString,
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
		Domain:   config.Cookie.Domain,
	}
	if !session {
		cookie.Expires = time.Now().Add(exp)
		cookie.MaxAge = config.Cookie.Duration
	}

	http.SetCookie(w, cookie)
	return nil
}

func (a *API) clearCookieTokens(config *conf.GlobalConfiguration, w http.ResponseWriter) {
	a.clearCookieToken(config, "access-token", w)
	a.clearCookieToken(config, "refresh-token", w)
}

func (a *API) clearCookieToken(config *conf.GlobalConfiguration, name string, w http.ResponseWriter) {
	cookieName := config.Cookie.Key
	if name != "" {
		cookieName += "-" + name
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour * 10),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
		Domain:   config.Cookie.Domain,
	})
}

func validateTokenClaims(outputClaims map[string]interface{}) error {
	schemaLoader := gojsonschema.NewStringLoader(hooks.MinimumViableTokenSchema)

	documentLoader := gojsonschema.NewGoLoader(outputClaims)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		var errorMessages string

		for _, desc := range result.Errors() {
			errorMessages += fmt.Sprintf("- %s\n", desc)
			fmt.Printf("- %s\n", desc)
		}
		return fmt.Errorf("output claims do not conform to the expected schema: \n%s", errorMessages)

	}

	return nil
}
