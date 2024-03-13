package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/supabase/auth/internal/api/sms_provider"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

// UserUpdateParams parameters for updating a user
type UserUpdateParams struct {
	Email               string                 `json:"email"`
	Password            *string                `json:"password"`
	Nonce               string                 `json:"nonce"`
	Data                map[string]interface{} `json:"data"`
	AppData             map[string]interface{} `json:"app_metadata,omitempty"`
	Phone               string                 `json:"phone"`
	Channel             string                 `json:"channel"`
	ResponseType        string                 `json:"response_type"`
	CodeChallenge       string                 `json:"code_challenge"`
	CodeChallengeMethod string                 `json:"code_challenge_method"`
}

func (a *API) validateUserUpdateParams(ctx context.Context, p *UserUpdateParams) error {
	config := a.config

	var err error
	if p.Email != "" {
		p.Email, err = validateEmail(p.Email)
		if err != nil {
			return err
		}
	}

	if p.Phone != "" {
		if p.Phone, err = validatePhone(p.Phone); err != nil {
			return err
		}
		if p.Channel == "" {
			p.Channel = sms_provider.SMSProvider
		}
		if !sms_provider.IsValidMessageChannel(p.Channel, config.Sms.Provider) {
			return badRequestError(InvalidChannelError)
		}
	}

	if p.Password != nil {
		if err := a.checkPasswordStrength(ctx, *p.Password); err != nil {
			return err
		}
	}

	return nil
}

// UserGet returns a user
func (a *API) UserGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		return badRequestError("Could not read claims")
	}

	aud := a.requestAud(ctx, r)
	if aud != claims.Audience {
		return badRequestError("Token audience doesn't match request audience")
	}

	user := getUser(ctx)
	return sendJSON(w, http.StatusOK, user)
}

// UserUpdate updates fields on a user
func (a *API) UserUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config
	aud := a.requestAud(ctx, r)

	params := &UserUpdateParams{}
	if err := retrieveRequestParams(r, params); err != nil {
		return err
	}

	user := getUser(ctx)
	session := getSession(ctx)

	if err := a.validateUserUpdateParams(ctx, params); err != nil {
		return err
	}

	if params.AppData != nil && !isAdmin(user, config) {
		if !isAdmin(user, config) {
			return unauthorizedError("Updating app_metadata requires admin privileges")
		}
	}

	if user.IsAnonymous {
		updatingForbiddenFields := false
		updatingForbiddenFields = updatingForbiddenFields || (params.Password != nil && *params.Password != "")
		if updatingForbiddenFields {
			return unprocessableEntityError("Updating password of an anonymous user is not possible")
		}
	}

	if user.IsSSOUser {
		updatingForbiddenFields := false

		updatingForbiddenFields = updatingForbiddenFields || (params.Password != nil && *params.Password != "")
		updatingForbiddenFields = updatingForbiddenFields || (params.Email != "" && params.Email != user.GetEmail())
		updatingForbiddenFields = updatingForbiddenFields || (params.Phone != "" && params.Phone != user.GetPhone())
		updatingForbiddenFields = updatingForbiddenFields || (params.Nonce != "")

		if updatingForbiddenFields {
			return unprocessableEntityError("Updating email, phone, password of a SSO account only possible via SSO")
		}
	}

	if params.Email != "" && user.GetEmail() != params.Email {
		if duplicateUser, err := models.IsDuplicatedEmail(db, params.Email, aud, user); err != nil {
			return internalServerError("Database error checking email").WithInternalError(err)
		} else if duplicateUser != nil {
			return unprocessableEntityError(DuplicateEmailMsg)
		}
	}

	if params.Phone != "" && user.GetPhone() != params.Phone {
		if exists, err := models.IsDuplicatedPhone(db, params.Phone, aud); err != nil {
			return internalServerError("Database error checking phone").WithInternalError(err)
		} else if exists {
			return unprocessableEntityError(DuplicatePhoneMsg)
		}
	}

	if params.Password != nil {
		if config.Security.UpdatePasswordRequireReauthentication {
			now := time.Now()
			// we require reauthentication if the user hasn't signed in recently in the current session
			if session == nil || now.After(session.CreatedAt.Add(24*time.Hour)) {
				if len(params.Nonce) == 0 {
					return badRequestError("Password update requires reauthentication")
				}
				if err := a.verifyReauthentication(params.Nonce, db, config, user); err != nil {
					return err
				}
			}
		}

		password := *params.Password
		if password != "" {
			if user.EncryptedPassword != "" && user.Authenticate(ctx, password) {
				return unprocessableEntityError("New password should be different from the old password.")
			}
		}

		if err := user.SetPassword(ctx, password); err != nil {
			return err
		}
	}

	err := db.Transaction(func(tx *storage.Connection) error {
		var terr error
		if params.Password != nil {
			var sessionID *uuid.UUID
			if session != nil {
				sessionID = &session.ID
			}

			if terr = user.UpdatePassword(tx, sessionID); terr != nil {
				return internalServerError("Error during password storage").WithInternalError(terr)
			}

			if terr := models.NewAuditLogEntry(r, tx, user, models.UserUpdatePasswordAction, "", nil); terr != nil {
				return terr
			}
		}

		if params.Data != nil {
			if terr = user.UpdateUserMetaData(tx, params.Data); terr != nil {
				return internalServerError("Error updating user").WithInternalError(terr)
			}
		}

		if params.AppData != nil {
			if terr = user.UpdateAppMetaData(tx, params.AppData); terr != nil {
				return internalServerError("Error updating user").WithInternalError(terr)
			}
		}

		if params.Email != "" && params.Email != user.GetEmail() {
			mailer := a.Mailer(ctx)
			referrer := utilities.GetReferrer(r, config)
			flowType := getFlow(params.CodeChallenge, params.ResponseType)
			if isCodeFlow(flowType) {
				_, terr := generateFlowState(tx, models.EmailChange.String(), models.EmailChange, params.CodeChallengeMethod, params.CodeChallenge, &user.ID, flowType)
				if terr != nil {
					return terr
				}

			}
			externalURL := getExternalHost(ctx)
			if terr = a.sendEmailChange(tx, config, user, mailer, params.Email, referrer, externalURL, config.Mailer.OtpLength, flowType); terr != nil {
				if errors.Is(terr, MaxFrequencyLimitError) {
					return tooManyRequestsError("For security purposes, you can only request this once every 60 seconds")
				}
				return internalServerError("Error sending change email").WithInternalError(terr)
			}
		}

		if params.Phone != "" && params.Phone != user.GetPhone() {
			if config.Sms.Autoconfirm {
				user.PhoneChange = params.Phone
				if _, terr := a.smsVerify(r, tx, user, &VerifyParams{
					Type:  phoneChangeVerification,
					Phone: params.Phone,
				}); terr != nil {
					return terr
				}
			} else {
				smsProvider, terr := sms_provider.GetSmsProvider(*config)
				if terr != nil {
					return badRequestError("Error sending sms: %v", terr)
				}
				if _, terr := a.sendPhoneConfirmation(tx, user, params.Phone, phoneChangeVerification, smsProvider, params.Channel); terr != nil {
					return internalServerError("Error sending phone change otp").WithInternalError(terr)
				}
			}
		}

		if terr = models.NewAuditLogEntry(r, tx, user, models.UserModifiedAction, "", nil); terr != nil {
			return internalServerError("Error recording audit log entry").WithInternalError(terr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, user)
}
