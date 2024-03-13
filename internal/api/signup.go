package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/structs"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/supabase/auth/internal/api/provider"
	"github.com/supabase/auth/internal/api/sms_provider"
	"github.com/supabase/auth/internal/metering"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

// SignupParams are the parameters the Signup endpoint accepts
type SignupParams struct {
	Email               string                 `json:"email"`
	Phone               string                 `json:"phone"`
	Password            string                 `json:"password"`
	Data                map[string]interface{} `json:"data"`
	Provider            string                 `json:"-"`
	Aud                 string                 `json:"-"`
	Channel             string                 `json:"channel"`
	CodeChallengeMethod string                 `json:"code_challenge_method"`
	CodeChallenge       string                 `json:"code_challenge"`
	ResponseType        string                 `json:"response_type"`
}

func (a *API) validateSignupParams(ctx context.Context, p *SignupParams) error {
	config := a.config

	if p.Password == "" {
		return unprocessableEntityError("Signup requires a valid password")
	}

	if err := a.checkPasswordStrength(ctx, p.Password); err != nil {
		return err
	}
	if p.Email != "" && p.Phone != "" {
		return unprocessableEntityError("Only an email address or phone number should be provided on signup.")
	}
	if p.Provider == "phone" && !sms_provider.IsValidMessageChannel(p.Channel, config.Sms.Provider) {
		return badRequestError(InvalidChannelError)
	}
	// PKCE not needed as phone signups already return access token in body
	if p.Phone != "" && p.CodeChallenge != "" {
		return badRequestError("PKCE not supported for phone signups")
	}
	if err := validateCodeFlowParams(p.CodeChallengeMethod, p.CodeChallenge, p.ResponseType); err != nil {
		return err
	}

	return nil
}

func (p *SignupParams) ConfigureDefaults() {
	if p.Email != "" {
		p.Provider = "email"
	} else if p.Phone != "" {
		p.Provider = "phone"
	}
	if p.Data == nil {
		p.Data = make(map[string]interface{})
	}

	// For backwards compatibility, we default to SMS if params Channel is not specified
	if p.Phone != "" && p.Channel == "" {
		p.Channel = sms_provider.SMSProvider
	}
}

func (params *SignupParams) ToUserModel(isSSOUser bool) (user *models.User, err error) {
	switch params.Provider {
	case "email":
		user, err = models.NewUser("", params.Email, params.Password, params.Aud, params.Data)
	case "phone":
		user, err = models.NewUser(params.Phone, "", params.Password, params.Aud, params.Data)
	case "anonymous":
		user, err = models.NewUser("", "", "", params.Aud, params.Data)
		user.IsAnonymous = true
	default:
		// handles external provider case
		user, err = models.NewUser("", params.Email, params.Password, params.Aud, params.Data)
	}
	if err != nil {
		err = internalServerError("Database error creating user").WithInternalError(err)
		return
	}
	user.IsSSOUser = isSSOUser
	if user.AppMetaData == nil {
		user.AppMetaData = make(map[string]interface{})
	}

	user.Identities = make([]models.Identity, 0)

	if params.Provider != "anonymous" {
		// TODO: Deprecate "provider" field
		user.AppMetaData["provider"] = params.Provider

		user.AppMetaData["providers"] = []string{params.Provider}
	}

	return user, nil
}

// Signup is the endpoint for registering a new user
func (a *API) Signup(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.config
	db := a.db.WithContext(ctx)

	if config.DisableSignup {
		return forbiddenError("Signups not allowed for this instance")
	}

	params := &SignupParams{}
	if err := retrieveRequestParams(r, params); err != nil {
		return err
	}

	params.ConfigureDefaults()

	if err := a.validateSignupParams(ctx, params); err != nil {
		return err
	}

	var err error
	flowType := getFlow(params.CodeChallenge, params.ResponseType)

	var user *models.User
	var grantParams models.GrantParams

	grantParams.FillGrantParams(r)

	params.Aud = a.requestAud(ctx, r)

	switch params.Provider {
	case "email":
		if !config.External.Email.Enabled {
			return badRequestError("Email signups are disabled")
		}
		params.Email, err = validateEmail(params.Email)
		if err != nil {
			return err
		}
		user, err = models.IsDuplicatedEmail(db, params.Email, params.Aud, nil)
	case "phone":
		if !config.External.Phone.Enabled {
			return badRequestError("Phone signups are disabled")
		}
		params.Phone, err = validatePhone(params.Phone)
		if err != nil {
			return err
		}
		user, err = models.FindUserByPhoneAndAudience(db, params.Phone, params.Aud)
	default:
		return invalidSignupError(config)
	}

	if err != nil && !models.IsNotFoundError(err) {
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	var signupUser *models.User
	if user == nil {
		// always call this outside of a database transaction as this method
		// can be computationally hard and block due to password hashing
		signupUser, err = params.ToUserModel(false /* <- isSSOUser */)
		if err != nil {
			return err
		}
	}

	err = db.Transaction(func(tx *storage.Connection) error {
		var terr error
		if user != nil {
			if (params.Provider == "email" && user.IsConfirmed()) || (params.Provider == "phone" && user.IsPhoneConfirmed()) {
				return UserExistsError
			}
			// do not update the user because we can't be sure of their claimed identity
		} else {
			user, terr = a.signupNewUser(tx, signupUser)
			if terr != nil {
				return terr
			}
		}
		identity, terr := models.FindIdentityByIdAndProvider(tx, user.ID.String(), "email")
		if terr != nil {
			if !models.IsNotFoundError(terr) {
				return terr
			}
			identityData := structs.Map(provider.Claims{
				Subject: user.ID.String(),
				Email:   user.GetEmail(),
			})
			for k, v := range params.Data {
				if _, ok := identityData[k]; !ok {
					identityData[k] = v
				}
			}
			identity, terr = a.createNewIdentity(tx, user, params.Provider, identityData)
			if terr != nil {
				return terr
			}
			if terr := user.RemoveUnconfirmedIdentities(tx, identity); terr != nil {
				return terr
			}
		}
		user.Identities = []models.Identity{*identity}

		if params.Provider == "email" && !user.IsConfirmed() {
			if config.Mailer.Autoconfirm {
				if terr = models.NewAuditLogEntry(r, tx, user, models.UserSignedUpAction, "", map[string]interface{}{
					"provider": params.Provider,
				}); terr != nil {
					return terr
				}
				if terr = user.Confirm(tx); terr != nil {
					return internalServerError("Database error updating user").WithInternalError(terr)
				}
			} else {
				mailer := a.Mailer(ctx)
				referrer := utilities.GetReferrer(r, config)
				if terr = models.NewAuditLogEntry(r, tx, user, models.UserConfirmationRequestedAction, "", map[string]interface{}{
					"provider": params.Provider,
				}); terr != nil {
					return terr
				}
				if isCodeFlow(flowType) {
					_, terr := generateFlowState(tx, params.Provider, models.EmailSignup, params.CodeChallengeMethod, params.CodeChallenge, &user.ID, flowType)
					if terr != nil {
						return terr
					}
				}
				externalURL := getExternalHost(ctx)
				if terr = sendConfirmation(tx, user, mailer, config.SMTP.MaxFrequency, referrer, externalURL, config.Mailer.OtpLength, flowType); terr != nil {
					if errors.Is(terr, MaxFrequencyLimitError) {
						now := time.Now()
						left := user.ConfirmationSentAt.Add(config.SMTP.MaxFrequency).Sub(now) / time.Second
						return tooManyRequestsError(fmt.Sprintf("For security purposes, you can only request this after %d seconds.", left))
					}
					return internalServerError("Error sending confirmation mail").WithInternalError(terr)
				}
			}
		} else if params.Provider == "phone" && !user.IsPhoneConfirmed() {
			if config.Sms.Autoconfirm {
				if terr = models.NewAuditLogEntry(r, tx, user, models.UserSignedUpAction, "", map[string]interface{}{
					"provider": params.Provider,
					"channel":  params.Channel,
				}); terr != nil {
					return terr
				}
				if terr = user.ConfirmPhone(tx); terr != nil {
					return internalServerError("Database error updating user").WithInternalError(terr)
				}
			} else {
				if terr = models.NewAuditLogEntry(r, tx, user, models.UserConfirmationRequestedAction, "", map[string]interface{}{
					"provider": params.Provider,
				}); terr != nil {
					return terr
				}
				smsProvider, terr := sms_provider.GetSmsProvider(*config)
				if terr != nil {
					return badRequestError("Error sending confirmation sms: %v", terr)
				}
				if _, terr := a.sendPhoneConfirmation(tx, user, params.Phone, phoneConfirmationOtp, smsProvider, params.Channel); terr != nil {
					return badRequestError("Error sending confirmation sms: %v", terr)
				}
			}
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, MaxFrequencyLimitError) {
			return tooManyRequestsError("For security purposes, you can only request this once every minute")
		}
		if errors.Is(err, UserExistsError) {
			err = db.Transaction(func(tx *storage.Connection) error {
				if terr := models.NewAuditLogEntry(r, tx, user, models.UserRepeatedSignUpAction, "", map[string]interface{}{
					"provider": params.Provider,
				}); terr != nil {
					return terr
				}
				return nil
			})
			if err != nil {
				return err
			}
			if config.Mailer.Autoconfirm || config.Sms.Autoconfirm {
				return badRequestError("User already registered")
			}
			sanitizedUser, err := sanitizeUser(user, params)
			if err != nil {
				return err
			}
			return sendJSON(w, http.StatusOK, sanitizedUser)
		}
		return err
	}

	// handles case where Mailer.Autoconfirm is true or Phone.Autoconfirm is true
	if user.IsConfirmed() || user.IsPhoneConfirmed() {
		var token *AccessTokenResponse
		err = db.Transaction(func(tx *storage.Connection) error {
			var terr error
			if terr = models.NewAuditLogEntry(r, tx, user, models.LoginAction, "", map[string]interface{}{
				"provider": params.Provider,
			}); terr != nil {
				return terr
			}
			token, terr = a.issueRefreshToken(ctx, tx, user, models.PasswordGrant, grantParams)

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
		metering.RecordLogin("password", user.ID)
		return sendJSON(w, http.StatusOK, token)
	}
	if user.HasBeenInvited() {
		// Remove sensitive fields
		user.UserMetaData = map[string]interface{}{}
		user.Identities = []models.Identity{}
	}
	return sendJSON(w, http.StatusOK, user)
}

// sanitizeUser removes all user sensitive information from the user object
// Should be used whenever we want to prevent information about whether a user is registered or not from leaking
func sanitizeUser(u *models.User, params *SignupParams) (*models.User, error) {
	now := time.Now()

	u.ID = uuid.Must(uuid.NewV4())

	u.Role = ""
	u.CreatedAt, u.UpdatedAt, u.ConfirmationSentAt = now, now, &now
	u.LastSignInAt, u.ConfirmedAt, u.EmailConfirmedAt, u.PhoneConfirmedAt = nil, nil, nil, nil
	u.Identities = make([]models.Identity, 0)
	u.UserMetaData = params.Data
	u.Aud = params.Aud

	// sanitize app_metadata
	u.AppMetaData = map[string]interface{}{
		"provider":  params.Provider,
		"providers": []string{params.Provider},
	}

	// sanitize param fields
	switch params.Provider {
	case "email":
		u.Phone = ""
	case "phone":
		u.Email = ""
	default:
		u.Phone, u.Email = "", ""
	}

	return u, nil
}

func (a *API) signupNewUser(conn *storage.Connection, user *models.User) (*models.User, error) {
	config := a.config

	err := conn.Transaction(func(tx *storage.Connection) error {
		var terr error
		if terr = tx.Create(user); terr != nil {
			return internalServerError("Database error saving new user").WithInternalError(terr)
		}
		if terr = user.SetRole(tx, config.JWT.DefaultGroupName); terr != nil {
			return internalServerError("Database error updating user").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// there may be triggers or generated column values in the database that will modify the
	// user data as it is being inserted. thus we load the user object
	// again to fetch those changes.
	if err := conn.Reload(user); err != nil {
		return nil, internalServerError("Database error loading user after sign-up").WithInternalError(err)
	}

	return user, nil
}
