package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/gofrs/uuid"
	jwt "github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
	"github.com/supabase/gotrue/internal/api/provider"
	"github.com/supabase/gotrue/internal/models"
	"github.com/supabase/gotrue/internal/observability"
	"github.com/supabase/gotrue/internal/storage"
	"github.com/supabase/gotrue/internal/utilities"
	"golang.org/x/oauth2"
)

// ExternalProviderClaims are the JWT claims sent as the state in the external oauth provider signup flow
type ExternalProviderClaims struct {
	AuthMicroserviceClaims
	Provider    string `json:"provider"`
	InviteToken string `json:"invite_token,omitempty"`
	Referrer    string `json:"referrer,omitempty"`
	FlowStateID string `json:"flow_state_id"`
}

// ExternalProviderRedirect redirects the request to the corresponding oauth provider
func (a *API) ExternalProviderRedirect(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config

	query := r.URL.Query()
	providerType := query.Get("provider")
	scopes := query.Get("scopes")
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")

	p, err := a.Provider(ctx, providerType, scopes)
	if err != nil {
		return badRequestError("Unsupported provider: %+v", err).WithInternalError(err)
	}

	inviteToken := query.Get("invite_token")
	if inviteToken != "" {
		_, userErr := models.FindUserByConfirmationToken(db, inviteToken)
		if userErr != nil {
			if models.IsNotFoundError(userErr) {
				return notFoundError(userErr.Error())
			}
			return internalServerError("Database error finding user").WithInternalError(userErr)
		}
	}

	redirectURL := utilities.GetReferrer(r, config)
	log := observability.GetLogEntry(r)
	log.WithField("provider", providerType).Info("Redirecting to external provider")
	if err := validatePKCEParams(codeChallengeMethod, codeChallenge); err != nil {
		return err
	}
	flowType := getFlowFromChallenge(codeChallenge)

	flowStateID := ""
	if flowType == models.PKCEFlow {
		codeChallengeMethodType, err := models.ParseCodeChallengeMethod(codeChallengeMethod)
		if err != nil {
			return err
		}
		flowState, err := models.NewFlowState(providerType, codeChallenge, codeChallengeMethodType, models.OAuth)
		if err != nil {
			return err
		}
		if err := a.db.Create(flowState); err != nil {
			return err
		}
		flowStateID = flowState.ID.String()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, ExternalProviderClaims{
		AuthMicroserviceClaims: AuthMicroserviceClaims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			},
			SiteURL:    config.SiteURL,
			InstanceID: uuid.Nil.String(),
		},
		Provider:    providerType,
		InviteToken: inviteToken,
		Referrer:    redirectURL,
		FlowStateID: flowStateID,
	})
	tokenString, err := token.SignedString([]byte(config.JWT.Secret))
	if err != nil {
		return internalServerError("Error creating state").WithInternalError(err)
	}

	authUrlParams := make([]oauth2.AuthCodeOption, 0)
	query.Del("scopes")
	query.Del("provider")
	query.Del("code_challenge")
	query.Del("code_challenge_method")
	for key := range query {
		if key == "workos_provider" {
			// See https://workos.com/docs/reference/sso/authorize/get
			authUrlParams = append(authUrlParams, oauth2.SetAuthURLParam("provider", query.Get(key)))
		} else {
			authUrlParams = append(authUrlParams, oauth2.SetAuthURLParam(key, query.Get(key)))
		}
	}

	var authURL string
	switch externalProvider := p.(type) {
	case *provider.TwitterProvider:
		authURL = externalProvider.AuthCodeURL(tokenString, authUrlParams...)
		err := storage.StoreInSession(providerType, externalProvider.Marshal(), r, w)
		if err != nil {
			return internalServerError("Error storing request token in session").WithInternalError(err)
		}
	default:
		authURL = p.AuthCodeURL(tokenString, authUrlParams...)
	}

	http.Redirect(w, r, authURL, http.StatusFound)
	return nil
}

// ExternalProviderCallback handles the callback endpoint in the external oauth provider flow
func (a *API) ExternalProviderCallback(w http.ResponseWriter, r *http.Request) error {
	rurl := a.getExternalRedirectURL(r)
	u, err := url.Parse(rurl)
	if err != nil {
		return err
	}
	a.redirectErrors(a.internalExternalProviderCallback, w, r, u)
	return nil
}

// errReturnNil is a hack that signals internalExternalProviderCallback to return nil
var errReturnNil = errors.New("createAccountFromExternalIdentity: return nil in internalExternalProviderCallback")

func (a *API) internalExternalProviderCallback(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config

	providerType := getExternalProviderType(ctx)
	var userData *provider.UserProvidedData
	var providerAccessToken string
	var providerRefreshToken string
	var grantParams models.GrantParams
	var err error

	if providerType == "twitter" {
		// future OAuth1.0 providers will use this method
		oAuthResponseData, err := a.oAuth1Callback(ctx, r, providerType)
		if err != nil {
			return err
		}
		userData = oAuthResponseData.userData
		providerAccessToken = oAuthResponseData.token
	} else {
		oAuthResponseData, err := a.oAuthCallback(ctx, r, providerType)
		if err != nil {
			return err
		}
		userData = oAuthResponseData.userData
		providerAccessToken = oAuthResponseData.token
		providerRefreshToken = oAuthResponseData.refreshToken
	}

	var flowState *models.FlowState
	// if there's a non-empty FlowStateID we perform PKCE Flow
	if flowStateID := getFlowStateID(ctx); flowStateID != "" {
		flowState, err = models.FindFlowStateByID(a.db, flowStateID)
		if err != nil {
			return err
		}
	}

	var user *models.User
	var token *AccessTokenResponse
	err = db.Transaction(func(tx *storage.Connection) error {
		var terr error
		inviteToken := getInviteToken(ctx)
		if inviteToken != "" {
			if user, terr = a.processInvite(r, ctx, tx, userData, inviteToken, providerType); terr != nil {
				return terr
			}
		} else {
			if user, terr = a.createAccountFromExternalIdentity(tx, r, userData, providerType); terr != nil {
				if errors.Is(terr, errReturnNil) {
					return nil
				}

				return terr
			}
		}
		if flowState != nil {
			// This means that the callback is using PKCE
			flowState.ProviderAccessToken = providerAccessToken
			flowState.ProviderRefreshToken = providerRefreshToken
			flowState.UserID = &(user.ID)
			terr = tx.Update(flowState)
		} else {
			token, terr = a.issueRefreshToken(ctx, tx, user, models.OAuth, grantParams)
		}

		if terr != nil {
			return oauthError("server_error", terr.Error())
		}
		return nil
	})
	if err != nil {
		return err
	}

	rurl := a.getExternalRedirectURL(r)
	if flowState != nil {
		// This means that the callback is using PKCE
		// Set the flowState.AuthCode to the query param here
		rurl, err = a.prepPKCERedirectURL(rurl, flowState.AuthCode)
		if err != nil {
			return err
		}
	} else if token != nil {
		q := url.Values{}
		q.Set("provider_token", providerAccessToken)
		// Because not all providers give out a refresh token
		// See corresponding OAuth2 spec: <https://www.rfc-editor.org/rfc/rfc6749.html#section-5.1>
		if providerRefreshToken != "" {
			q.Set("provider_refresh_token", providerRefreshToken)
		}

		rurl = token.AsRedirectURL(rurl, q)

		if err := a.setCookieTokens(config, token, false, w); err != nil {
			return internalServerError("Failed to set JWT cookie. %s", err)
		}
	} else {
		// Left as hash fragment to comply with spec. Additionally, may override existing error query param if set to PKCE.
		rurl, err = a.prepErrorRedirectURL(unauthorizedError("Unverified email with %v", providerType), w, r, rurl, models.ImplicitFlow)
		if err != nil {
			return err
		}
	}

	http.Redirect(w, r, rurl, http.StatusFound)
	return nil
}

func (a *API) createAccountFromExternalIdentity(tx *storage.Connection, r *http.Request, userData *provider.UserProvidedData, providerType string) (*models.User, error) {
	ctx := r.Context()
	aud := a.requestAud(ctx, r)
	config := a.config

	var terr error

	var user *models.User
	var identity *models.Identity

	var emailData provider.Email
	var identityData map[string]interface{}
	if userData.Metadata != nil {
		identityData = structs.Map(userData.Metadata)
	}

	var emails []string

	for _, email := range userData.Emails {
		if email.Verified || config.Mailer.Autoconfirm {
			emails = append(emails, strings.ToLower(email.Email))
		}
	}

	decision, terr := models.DetermineAccountLinking(tx, providerType, userData.Metadata.Subject, emails)
	if terr != nil {
		return nil, terr
	}

	switch decision.Decision {
	case models.LinkAccount:
		user = decision.User

		emailData = userData.Emails[0]
		for _, e := range userData.Emails {
			if e.Primary || e.Verified {
				emailData = e
				break
			}
		}

		if _, terr = a.createNewIdentity(tx, user, providerType, identityData); terr != nil {
			return nil, terr
		}

		if terr = user.UpdateAppMetaDataProviders(tx); terr != nil {
			return nil, terr
		}

	case models.CreateAccount:
		if config.DisableSignup {
			return nil, forbiddenError("Signups not allowed for this instance")
		}

		// prefer primary email for new signups
		emailData = userData.Emails[0]
		for _, e := range userData.Emails {
			if e.Primary {
				emailData = e
				break
			}
		}

		params := &SignupParams{
			Provider: providerType,
			Email:    emailData.Email,
			Aud:      aud,
			Data:     identityData,
		}

		isSSOUser := strings.HasPrefix(providerType, "sso:")

		user, terr = a.signupNewUser(ctx, tx, params, isSSOUser)
		if terr != nil {
			return nil, terr
		}

		if _, terr = a.createNewIdentity(tx, user, providerType, identityData); terr != nil {
			return nil, terr
		}

	case models.AccountExists:
		user = decision.User
		identity = decision.Identities[0]

		identity.IdentityData = identityData
		if terr = tx.UpdateOnly(identity, "identity_data", "last_sign_in_at"); terr != nil {
			return nil, terr
		}
		// email & verified status might have changed if identity's email changed
		emailData = provider.Email{
			Email:    userData.Metadata.Email,
			Verified: userData.Metadata.EmailVerified,
		}
		if terr = user.UpdateUserMetaData(tx, identityData); terr != nil {
			return nil, terr
		}
		if terr = user.UpdateAppMetaDataProviders(tx); terr != nil {
			return nil, terr
		}

	case models.MultipleAccounts:
		return nil, internalServerError(fmt.Sprintf("Multiple accounts with the same email address in the same linking domain detected: %v", decision.LinkingDomain))

	default:
		return nil, internalServerError(fmt.Sprintf("Unknown automatic linking decision: %v", decision.Decision))
	}

	if user.IsBanned() {
		return nil, unauthorizedError("User is unauthorized")
	}

	// an account with a previously unconfirmed email + password
	// combination or phone may exist. so now that there is an
	// OAuth identity bound to this user, and since they have not
	// confirmed their email or phone, they are unaware that a
	// potentially malicious door exists into their account; thus
	// the password and phone needs to be removed.
	if terr = user.RemoveUnconfirmedIdentities(tx); terr != nil {
		return nil, internalServerError("Error updating user").WithInternalError(terr)
	}

	if !user.IsConfirmed() {
		if !emailData.Verified && !config.Mailer.Autoconfirm {
			mailer := a.Mailer(ctx)
			referrer := utilities.GetReferrer(r, config)
			externalURL := getExternalHost(ctx)
			if terr = sendConfirmation(tx, user, mailer, config.SMTP.MaxFrequency, referrer, externalURL, config.Mailer.OtpLength, models.ImplicitFlow); terr != nil {
				if errors.Is(terr, MaxFrequencyLimitError) {
					return nil, tooManyRequestsError("For security purposes, you can only request this once every minute")
				}
				return nil, internalServerError("Error sending confirmation mail").WithInternalError(terr)
			}
			// email must be verified to issue a token
			return nil, errReturnNil
		}

		if terr := models.NewAuditLogEntry(r, tx, user, models.UserSignedUpAction, "", map[string]interface{}{
			"provider": providerType,
		}); terr != nil {
			return nil, terr
		}
		if terr = triggerEventHooks(ctx, tx, SignupEvent, user, config); terr != nil {
			return nil, terr
		}

		// fall through to auto-confirm and issue token
		if terr = user.Confirm(tx); terr != nil {
			return nil, internalServerError("Error updating user").WithInternalError(terr)
		}
	} else {
		if terr := models.NewAuditLogEntry(r, tx, user, models.LoginAction, "", map[string]interface{}{
			"provider": providerType,
		}); terr != nil {
			return nil, terr
		}
		if terr = triggerEventHooks(ctx, tx, LoginEvent, user, config); terr != nil {
			return nil, terr
		}
	}

	return user, nil
}

func (a *API) processInvite(r *http.Request, ctx context.Context, tx *storage.Connection, userData *provider.UserProvidedData, inviteToken, providerType string) (*models.User, error) {
	config := a.config
	user, err := models.FindUserByConfirmationToken(tx, inviteToken)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(err.Error())
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	var emailData *provider.Email
	var emails []string
	for i, e := range userData.Emails {
		emails = append(emails, e.Email)
		if user.GetEmail() == e.Email {
			emailData = &userData.Emails[i]
			break
		}
	}

	if emailData == nil {
		return nil, badRequestError("Invited email does not match emails from external provider").WithInternalMessage("invited=%s external=%s", user.Email, strings.Join(emails, ", "))
	}

	var identityData map[string]interface{}
	if userData.Metadata != nil {
		identityData = structs.Map(userData.Metadata)
	}
	if _, err := a.createNewIdentity(tx, user, providerType, identityData); err != nil {
		return nil, err
	}
	if err = user.UpdateAppMetaData(tx, map[string]interface{}{
		"provider": providerType,
	}); err != nil {
		return nil, err
	}
	if err = user.UpdateAppMetaDataProviders(tx); err != nil {
		return nil, err
	}
	if err := user.UpdateUserMetaData(tx, identityData); err != nil {
		return nil, internalServerError("Database error updating user").WithInternalError(err)
	}

	if err := models.NewAuditLogEntry(r, tx, user, models.InviteAcceptedAction, "", map[string]interface{}{
		"provider": providerType,
	}); err != nil {
		return nil, err
	}
	if err := triggerEventHooks(ctx, tx, SignupEvent, user, config); err != nil {
		return nil, err
	}

	// an account with a previously unconfirmed email + password
	// combination or phone may exist. so now that there is an
	// OAuth identity bound to this user, and since they have not
	// confirmed their email or phone, they are unaware that a
	// potentially malicious door exists into their account; thus
	// the password and phone needs to be removed.
	if err = user.RemoveUnconfirmedIdentities(tx); err != nil {
		return nil, internalServerError("Error updating user").WithInternalError(err)
	}

	// confirm because they were able to respond to invite email
	if err := user.Confirm(tx); err != nil {
		return nil, err
	}
	return user, nil
}

func (a *API) loadExternalState(ctx context.Context, state string) (context.Context, error) {
	config := a.config
	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err := p.ParseWithClaims(state, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT.Secret), nil
	})
	if err != nil || claims.Provider == "" {
		return nil, badRequestError("OAuth state is invalid: %v", err)
	}
	if claims.InviteToken != "" {
		ctx = withInviteToken(ctx, claims.InviteToken)
	}
	if claims.Referrer != "" {
		ctx = withExternalReferrer(ctx, claims.Referrer)
	}
	if claims.FlowStateID != "" {
		ctx = withFlowStateID(ctx, claims.FlowStateID)
	}
	ctx = withExternalProviderType(ctx, claims.Provider)
	return withSignature(ctx, state), nil
}

// Provider returns a Provider interface for the given name.
func (a *API) Provider(ctx context.Context, name string, scopes string) (provider.Provider, error) {
	config := a.config
	name = strings.ToLower(name)

	switch name {
	case "apple":
		return provider.NewAppleProvider(ctx, config.External.Apple)
	case "azure":
		return provider.NewAzureProvider(config.External.Azure, scopes)
	case "bitbucket":
		return provider.NewBitbucketProvider(config.External.Bitbucket)
	case "discord":
		return provider.NewDiscordProvider(config.External.Discord, scopes)
	case "facebook":
		return provider.NewFacebookProvider(config.External.Facebook, scopes)
	case "figma":
		return provider.NewFigmaProvider(config.External.Figma, scopes)
	case "github":
		return provider.NewGithubProvider(config.External.Github, scopes)
	case "gitlab":
		return provider.NewGitlabProvider(config.External.Gitlab, scopes)
	case "google":
		return provider.NewGoogleProvider(ctx, config.External.Google, scopes)
	case "kakao":
		return provider.NewKakaoProvider(config.External.Kakao, scopes)
	case "keycloak":
		return provider.NewKeycloakProvider(config.External.Keycloak, scopes)
	case "linkedin":
		return provider.NewLinkedinProvider(config.External.Linkedin, scopes)
	case "naver":
		return provider.NewNaverProvider(config.External.Naver, scopes)
	case "notion":
		return provider.NewNotionProvider(config.External.Notion)
	case "spotify":
		return provider.NewSpotifyProvider(config.External.Spotify, scopes)
	case "slack":
		return provider.NewSlackProvider(config.External.Slack, scopes)
	case "twitch":
		return provider.NewTwitchProvider(config.External.Twitch, scopes)
	case "twitter":
		return provider.NewTwitterProvider(config.External.Twitter, scopes)
	case "workos":
		return provider.NewWorkOSProvider(config.External.WorkOS)
	case "zoom":
		return provider.NewZoomProvider(config.External.Zoom)
	default:
		return nil, fmt.Errorf("Provider %s could not be found", name)
	}
}

func (a *API) redirectErrors(handler apiHandler, w http.ResponseWriter, r *http.Request, u *url.URL) {
	ctx := r.Context()
	log := observability.GetLogEntry(r)
	errorID := getRequestID(ctx)
	err := handler(w, r)
	if err != nil {
		q := getErrorQueryString(err, errorID, log, u.Query())
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

func getErrorQueryString(err error, errorID string, log logrus.FieldLogger, q url.Values) *url.Values {
	switch e := err.(type) {
	case *HTTPError:
		if str, ok := oauthErrorMap[e.Code]; ok {
			q.Set("error", str)
		} else {
			q.Set("error", "server_error")
		}
		if e.Code >= http.StatusInternalServerError {
			e.ErrorID = errorID
			// this will get us the stack trace too
			log.WithError(e.Cause()).Error(e.Error())
		} else {
			log.WithError(e.Cause()).Info(e.Error())
		}
		q.Set("error_description", e.Message)
	case *OAuthError:
		q.Set("error", e.Err)
		q.Set("error_description", e.Description)
		log.WithError(e.Cause()).Info(e.Error())
	case ErrorCause:
		return getErrorQueryString(e.Cause(), errorID, log, q)
	default:
		error_type, error_description := "server_error", err.Error()

		// Provide better error messages for certain user-triggered Postgres errors.
		if pgErr := utilities.NewPostgresError(e); pgErr != nil {
			error_description = pgErr.Message
			if oauthErrorType, ok := oauthErrorMap[pgErr.HttpStatusCode]; ok {
				error_type = oauthErrorType
			}
		}

		q.Set("error", error_type)
		q.Set("error_description", error_description)
	}
	return &q
}

func (a *API) getExternalRedirectURL(r *http.Request) string {
	ctx := r.Context()
	config := a.config
	if config.External.RedirectURL != "" {
		return config.External.RedirectURL
	}
	if er := getExternalReferrer(ctx); er != "" {
		return er
	}
	return config.SiteURL
}

func (a *API) createNewIdentity(tx *storage.Connection, user *models.User, providerType string, identityData map[string]interface{}) (*models.Identity, error) {
	identity, err := models.NewIdentity(user, providerType, identityData)
	if err != nil {
		return nil, err
	}

	if terr := tx.Create(identity); terr != nil {
		return nil, internalServerError("Error creating identity").WithInternalError(terr)
	}

	return identity, nil
}
