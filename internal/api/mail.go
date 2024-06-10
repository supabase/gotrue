package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/supabase/auth/internal/hooks"
	mail "github.com/supabase/auth/internal/mailer"
	"golang.org/x/sync/errgroup"

	"github.com/badoux/checkmail"
	"github.com/fatih/structs"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	"github.com/supabase/auth/internal/api/provider"
	"github.com/supabase/auth/internal/crypto"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

var (
	MaxFrequencyLimitError error = errors.New("frequency limit reached")
)

type GenerateLinkParams struct {
	Type       string                 `json:"type"`
	Email      string                 `json:"email"`
	NewEmail   string                 `json:"new_email"`
	Password   string                 `json:"password"`
	Data       map[string]interface{} `json:"data"`
	RedirectTo string                 `json:"redirect_to"`
}

type GenerateLinkResponse struct {
	models.User
	ActionLink       string `json:"action_link"`
	EmailOtp         string `json:"email_otp"`
	HashedToken      string `json:"hashed_token"`
	VerificationType string `json:"verification_type"`
	RedirectTo       string `json:"redirect_to"`
}

func (a *API) adminGenerateLink(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config
	mailer := a.Mailer()
	adminUser := getAdminUser(ctx)
	params := &GenerateLinkParams{}
	if err := retrieveRequestParams(r, params); err != nil {
		return err
	}

	var err error
	params.Email, err = validateEmail(params.Email)
	if err != nil {
		return err
	}
	referrer := utilities.GetReferrer(r, config)
	if utilities.IsRedirectURLValid(config, params.RedirectTo) {
		referrer = params.RedirectTo
	}

	aud := a.requestAud(ctx, r)
	user, err := models.FindUserByEmailAndAudience(db, params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			switch params.Type {
			case mail.MagicLinkVerification:
				params.Type = mail.SignupVerification
				params.Password, err = password.Generate(64, 10, 1, false, true)
				if err != nil {
					// password generation must always succeed
					panic(err)
				}
			case mail.RecoveryVerification, mail.EmailChangeCurrentVerification, mail.EmailChangeNewVerification:
				return notFoundError(ErrorCodeUserNotFound, "User with this email not found")
			}
		} else {
			return internalServerError("Database error finding user").WithInternalError(err)
		}
	}

	var url string
	now := time.Now()
	otp, err := crypto.GenerateOtp(config.Mailer.OtpLength)
	if err != nil {
		// OTP generation must always succeed
		panic(err)
	}

	hashedToken := crypto.GenerateTokenHash(params.Email, otp)

	var signupUser *models.User
	if params.Type == mail.SignupVerification && user == nil {
		signupParams := &SignupParams{
			Email:    params.Email,
			Password: params.Password,
			Data:     params.Data,
			Provider: "email",
			Aud:      aud,
		}

		if err := a.validateSignupParams(ctx, signupParams); err != nil {
			return err
		}

		signupUser, err = signupParams.ToUserModel(false /* <- isSSOUser */)
		if err != nil {
			return err
		}
	}

	var ott *models.OneTimeToken
	err = db.Transaction(func(tx *storage.Connection) error {
		var terr error
		switch params.Type {
		case mail.MagicLinkVerification, mail.RecoveryVerification:
			if terr = models.NewAuditLogEntry(r, tx, user, models.UserRecoveryRequestedAction, "", nil); terr != nil {
				return terr
			}
			user.RecoverySentAt = &now
			terr = tx.UpdateOnly(user, "recovery_sent_at")
			if terr != nil {
				terr = errors.Wrap(terr, "Database error updating user for recovery")
				return terr
			}

			ott, terr = models.CreateOneTimeToken(tx, user.ID, user.GetEmail(), hashedToken, models.RecoveryToken)
			if terr != nil {
				terr = errors.Wrap(terr, "Database error creating recovery token in admin")
				return terr
			}
		case mail.InviteVerification:
			if user != nil {
				if user.IsConfirmed() {
					return unprocessableEntityError(ErrorCodeEmailExists, DuplicateEmailMsg)
				}
			} else {
				signupParams := &SignupParams{
					Email:    params.Email,
					Data:     params.Data,
					Provider: "email",
					Aud:      aud,
				}

				// because params above sets no password, this
				// method is not computationally hard so it can
				// be used within a database transaction
				user, terr = signupParams.ToUserModel(false /* <- isSSOUser */)
				if terr != nil {
					return terr
				}

				user, terr = a.signupNewUser(tx, user)
				if terr != nil {
					return terr
				}
				identity, terr := a.createNewIdentity(tx, user, "email", structs.Map(provider.Claims{
					Subject: user.ID.String(),
					Email:   user.GetEmail(),
				}))
				if terr != nil {
					return terr
				}
				user.Identities = []models.Identity{*identity}
			}
			if terr = models.NewAuditLogEntry(r, tx, adminUser, models.UserInvitedAction, "", map[string]interface{}{
				"user_id":    user.ID,
				"user_email": user.Email,
			}); terr != nil {
				return terr
			}
			user.ConfirmationSentAt = &now
			user.InvitedAt = &now
			terr = tx.UpdateOnly(user, "confirmation_sent_at", "invited_at")
			if terr != nil {
				terr = errors.Wrap(terr, "Database error updating user for invite")
				return terr
			}
			ott, terr = models.CreateOneTimeToken(tx, user.ID, user.GetEmail(), hashedToken, models.ConfirmationToken)
			if terr != nil {
				terr = errors.Wrap(terr, "Database error creating confirmation token for invite in admin")
				return terr
			}
		case mail.SignupVerification:
			if user != nil {
				if user.IsConfirmed() {
					return unprocessableEntityError(ErrorCodeEmailExists, DuplicateEmailMsg)
				}
				if err := user.UpdateUserMetaData(tx, params.Data); err != nil {
					return internalServerError("Database error updating user").WithInternalError(err)
				}
			} else {
				// you should never use SignupParams with
				// password here to generate a new user, use
				// signupUser which is a model generated from
				// SignupParams above
				user, terr = a.signupNewUser(tx, signupUser)
				if terr != nil {
					return terr
				}
				identity, terr := a.createNewIdentity(tx, user, "email", structs.Map(provider.Claims{
					Subject: user.ID.String(),
					Email:   user.GetEmail(),
				}))
				if terr != nil {
					return terr
				}
				user.Identities = []models.Identity{*identity}
			}
			user.ConfirmationSentAt = &now
			terr = tx.UpdateOnly(user, "confirmation_sent_at")
			if terr != nil {
				terr = errors.Wrap(terr, "Database error updating user for confirmation")
				return terr
			}
			ott, terr = models.CreateOneTimeToken(tx, user.ID, user.GetEmail(), hashedToken, models.ConfirmationToken)
			if terr != nil {
				terr = errors.Wrap(terr, "Database error creating confirmation token for signup in admin")
				return terr
			}
		case mail.EmailChangeCurrentVerification:
			if !config.Mailer.SecureEmailChangeEnabled {
				return badRequestError(ErrorCodeValidationFailed, "Enable secure email change to generate link for current email")
			}
			params.NewEmail, terr = validateEmail(params.NewEmail)
			if terr != nil {
				return terr
			}
			if duplicateUser, terr := models.IsDuplicatedEmail(tx, params.NewEmail, user.Aud, user); terr != nil {
				return internalServerError("Database error checking email").WithInternalError(terr)
			} else if duplicateUser != nil {
				return unprocessableEntityError(ErrorCodeEmailExists, DuplicateEmailMsg)
			}
			now := time.Now()
			user.EmailChangeSentAt = &now
			user.EmailChange = params.NewEmail
			user.EmailChangeConfirmStatus = zeroConfirmation
			if terr := tx.UpdateOnly(user, "email_change", "email_change_sent_at", "email_change_confirm_status"); terr != nil {
				return errors.Wrap(terr, "Database error updating user for email change")
			}
			ott, terr = models.CreateOneTimeToken(tx, user.ID, user.GetEmail(), hashedToken, models.EmailChangeTokenCurrent)
			if terr != nil {
				return errors.Wrap(terr, "Database error creating email change token current in admin")
			}
		case mail.EmailChangeNewVerification:
			params.NewEmail, terr = validateEmail(params.NewEmail)
			if terr != nil {
				return terr
			}
			if duplicateUser, terr := models.IsDuplicatedEmail(tx, params.NewEmail, user.Aud, user); terr != nil {
				return internalServerError("Database error checking email").WithInternalError(terr)
			} else if duplicateUser != nil {
				return unprocessableEntityError(ErrorCodeEmailExists, DuplicateEmailMsg)
			}
			now := time.Now()
			user.EmailChangeSentAt = &now
			user.EmailChange = params.NewEmail
			user.EmailChangeConfirmStatus = zeroConfirmation
			if terr := tx.UpdateOnly(user, "email_change", "email_change_sent_at", "email_change_confirm_status"); terr != nil {
				return errors.Wrap(terr, "Database error updating user for email change")
			}
			ott, terr = models.CreateOneTimeToken(tx, user.ID, user.EmailChange, crypto.GenerateTokenHash(params.NewEmail, otp), models.EmailChangeTokenNew)
			if terr != nil {
				return errors.Wrap(terr, "Database error creating email change token new in admin")
			}
		default:
			return badRequestError(ErrorCodeValidationFailed, "Invalid email action link type requested: %v", params.Type)
		}

		if terr != nil {
			return terr
		}

		externalURL := getExternalHost(ctx)
		url, terr = mailer.GetEmailActionLink(ott, params.Type, referrer, externalURL)
		if terr != nil {
			return internalServerError("Error generating email action link").WithInternalError(terr)
		}
		return nil
	})

	if err != nil {
		return err
	}

	resp := GenerateLinkResponse{
		User:             *user,
		ActionLink:       url,
		EmailOtp:         otp,
		HashedToken:      hashedToken,
		VerificationType: params.Type,
		RedirectTo:       referrer,
	}

	return sendJSON(w, http.StatusOK, resp)
}

func (a *API) sendConfirmation(r *http.Request, tx *storage.Connection, u *models.User, flowType models.FlowType) error {
	config := a.config
	maxFrequency := config.SMTP.MaxFrequency
	otpLength := config.Mailer.OtpLength

	if err := validateSentWithinFrequencyLimit(u.ConfirmationSentAt, maxFrequency); err != nil {
		return err
	}
	otp, err := crypto.GenerateOtp(otpLength)
	if err != nil {
		// OTP generation must succeeed
		panic(err)
	}
	token := addFlowPrefixToToken(crypto.GenerateTokenHash(u.GetEmail(), otp), flowType)
	ott, err := models.CreateOneTimeToken(tx, u.ID, u.GetEmail(), token, models.ConfirmationToken)
	if err != nil {
		return errors.Wrap(err, "Database error creating confirmation token")
	}
	now := time.Now()
	u.ConfirmationSentAt = &now
	if err := tx.UpdateOnly(u, "confirmation_sent_at"); err != nil {
		return errors.Wrap(err, "Database error updating user for confirmation")
	}
	if err := a.sendEmail(r, tx, u, ott, mail.SignupVerification, otp); err != nil {
		return errors.Wrap(err, "Error sending confirmation email")
	}
	return nil
}

func (a *API) sendInvite(r *http.Request, tx *storage.Connection, u *models.User) error {
	config := a.config
	otpLength := config.Mailer.OtpLength
	otp, err := crypto.GenerateOtp(otpLength)
	if err != nil {
		// OTP generation must succeed
		panic(err)
	}
	token := crypto.GenerateTokenHash(u.GetEmail(), otp)
	ott, err := models.CreateOneTimeToken(tx, u.ID, u.GetEmail(), token, models.ConfirmationToken)
	if err != nil {
		return errors.Wrap(err, "Database error creating confirmation token for invite")
	}
	now := time.Now()
	u.InvitedAt = &now
	u.ConfirmationSentAt = &now
	if err := tx.UpdateOnly(u, "confirmation_sent_at", "invited_at"); err != nil {
		return errors.Wrap(err, "Database error updating user for invite")
	}
	if err := a.sendEmail(r, tx, u, ott, mail.InviteVerification, otp); err != nil {
		return errors.Wrap(err, "Error sending invite email")
	}

	return nil
}

func (a *API) sendPasswordRecovery(r *http.Request, tx *storage.Connection, u *models.User, flowType models.FlowType) error {
	config := a.config
	maxFrequency := config.SMTP.MaxFrequency
	otpLength := config.Mailer.OtpLength
	if err := validateSentWithinFrequencyLimit(u.RecoverySentAt, maxFrequency); err != nil {
		return err
	}
	otp, err := crypto.GenerateOtp(otpLength)
	if err != nil {
		// OTP generation must succeed
		panic(err)
	}
	token := addFlowPrefixToToken(crypto.GenerateTokenHash(u.GetEmail(), otp), flowType)
	ott, err := models.CreateOneTimeToken(tx, u.ID, u.GetEmail(), token, models.RecoveryToken)
	if err != nil {
		return errors.Wrap(err, "Database error creating recovery token")
	}
	now := time.Now()
	u.RecoverySentAt = &now
	err = tx.UpdateOnly(u, "recovery_sent_at")
	if err != nil {
		return errors.Wrap(err, "Database error updating user for recovery")
	}
	if err = a.sendEmail(r, tx, u, ott, mail.RecoveryVerification, otp); err != nil {
		return errors.Wrap(err, "Error sending recovery email")
	}

	return nil
}

func (a *API) sendReauthenticationOtp(r *http.Request, tx *storage.Connection, u *models.User) error {
	config := a.config
	maxFrequency := config.SMTP.MaxFrequency
	otpLength := config.Mailer.OtpLength
	if err := validateSentWithinFrequencyLimit(u.ReauthenticationSentAt, maxFrequency); err != nil {
		return err
	}
	otp, err := crypto.GenerateOtp(otpLength)
	if err != nil {
		// OTP generation must succeed
		panic(err)
	}
	token := crypto.GenerateTokenHash(u.GetEmail(), otp)
	ott, err := models.CreateOneTimeToken(tx, u.ID, u.GetEmail(), token, models.ReauthenticationToken)
	if err != nil {
		return errors.Wrap(err, "Database error creating reauthentication token")
	}
	now := time.Now()
	u.ReauthenticationSentAt = &now
	if err := tx.UpdateOnly(u, "reauthentication_sent_at"); err != nil {
		return errors.Wrap(err, "Database error updating user for reauthentication")
	}
	if err := a.sendEmail(r, tx, u, ott, mail.ReauthenticationVerification, otp); err != nil {
		return errors.Wrap(err, "Error sending reauthentication email")
	}
	return nil
}

func (a *API) sendMagicLink(r *http.Request, tx *storage.Connection, u *models.User, flowType models.FlowType) error {
	config := a.config
	otpLength := config.Mailer.OtpLength
	maxFrequency := config.SMTP.MaxFrequency
	var err error
	// since Magic Link is just a recovery with a different template and behaviour
	// around new users we will reuse the recovery db timer to prevent potential abuse
	if err := validateSentWithinFrequencyLimit(u.RecoverySentAt, maxFrequency); err != nil {
		return err
	}
	otp, err := crypto.GenerateOtp(otpLength)
	if err != nil {
		// OTP generation must succeed
		panic(err)
	}
	token := addFlowPrefixToToken(crypto.GenerateTokenHash(u.GetEmail(), otp), flowType)
	ott, err := models.CreateOneTimeToken(tx, u.ID, u.GetEmail(), token, models.RecoveryToken)
	if err != nil {
		return errors.Wrap(err, "Database error creating recovery token")
	}
	now := time.Now()
	u.RecoverySentAt = &now
	if err = tx.UpdateOnly(u, "recovery_sent_at"); err != nil {
		return errors.Wrap(err, "Database error updating user for recovery")
	}
	if err := a.sendEmail(r, tx, u, ott, mail.MagicLinkVerification, otp); err != nil {
		return errors.Wrap(err, "Error sending magic link email")
	}
	return nil
}

// sendEmailChange sends out an email change token to the new email.
func (a *API) sendEmailChange(r *http.Request, tx *storage.Connection, u *models.User, email string, flowType models.FlowType) error {
	config := a.config
	otpLength := config.Mailer.OtpLength
	if err := validateSentWithinFrequencyLimit(u.EmailChangeSentAt, config.SMTP.MaxFrequency); err != nil {
		return err
	}

	otpNew, err := crypto.GenerateOtp(otpLength)
	if err != nil {
		// OTP generation must succeed
		panic(err)
	}
	u.EmailChange = email
	token := addFlowPrefixToToken(crypto.GenerateTokenHash(u.EmailChange, otpNew), flowType)
	ottNew, err := models.CreateOneTimeToken(tx, u.ID, u.EmailChange, token, models.EmailChangeTokenNew)
	if err != nil {
		return errors.Wrap(err, "Database error creating email change token new")
	}

	otpCurrent := ""
	var ottCurrent *models.OneTimeToken
	if config.Mailer.SecureEmailChangeEnabled && u.GetEmail() != "" {
		otpCurrent, err = crypto.GenerateOtp(otpLength)
		if err != nil {
			// OTP generation must succeed
			panic(err)
		}
		currentToken := addFlowPrefixToToken(crypto.GenerateTokenHash(u.GetEmail(), otpCurrent), flowType)
		ottCurrent, err = models.CreateOneTimeToken(tx, u.ID, u.GetEmail(), currentToken, models.EmailChangeTokenCurrent)
		if err != nil {
			return errors.Wrap(err, "Database error creating email change token current")
		}
	}

	now := time.Now()
	u.EmailChangeConfirmStatus = zeroConfirmation
	u.EmailChangeSentAt = &now
	if err := tx.UpdateOnly(
		u,
		"email_change",
		"email_change_sent_at",
		"email_change_confirm_status",
	); err != nil {
		return errors.Wrap(err, "Database error updating user for email change")
	}

	if err = a.sendEmailChangeEmails(r, tx, u, ottNew, ottCurrent, mail.EmailChangeVerification, otpCurrent, otpNew); err != nil {
		return err
	}

	return nil
}

func validateEmail(email string) (string, error) {
	if email == "" {
		return "", badRequestError(ErrorCodeValidationFailed, "An email address is required")
	}
	if len(email) > 255 {
		return "", badRequestError(ErrorCodeValidationFailed, "An email address is too long")
	}
	if err := checkmail.ValidateFormat(email); err != nil {
		return "", badRequestError(ErrorCodeValidationFailed, "Unable to validate email address: "+err.Error())
	}
	return strings.ToLower(email), nil
}

func validateSentWithinFrequencyLimit(sentAt *time.Time, frequency time.Duration) error {
	if sentAt != nil && sentAt.Add(frequency).After(time.Now()) {
		return MaxFrequencyLimitError
	}
	return nil
}

func (a *API) sendEmail(r *http.Request, tx *storage.Connection, u *models.User, ott *models.OneTimeToken, emailActionType, otp string) error {
	mailer := a.Mailer()
	ctx := r.Context()
	config := a.config
	referrerURL := utilities.GetReferrer(r, config)
	externalURL := getExternalHost(ctx)
	if config.Hook.SendEmail.Enabled {
		emailData := mail.EmailData{
			Token:           otp,
			EmailActionType: emailActionType,
			RedirectTo:      referrerURL,
			SiteURL:         externalURL.String(),
			TokenHash:       ott.TokenHash,
		}
		input := hooks.SendEmailInput{
			User:      u,
			EmailData: emailData,
		}
		output := hooks.SendEmailOutput{}
		return a.invokeHook(tx, r, &input, &output, config.Hook.SendEmail.URI)
	}

	switch emailActionType {
	case mail.SignupVerification:
		return mailer.ConfirmationMail(r, u, ott, otp, referrerURL, externalURL)
	case mail.MagicLinkVerification:
		return mailer.MagicLinkMail(r, u, ott, otp, referrerURL, externalURL)
	case mail.ReauthenticationVerification:
		return mailer.ReauthenticateMail(r, u, otp)
	case mail.RecoveryVerification:
		return mailer.RecoveryMail(r, u, ott, otp, referrerURL, externalURL)
	case mail.InviteVerification:
		return mailer.InviteMail(r, u, ott, otp, referrerURL, externalURL)
	default:
		return errors.New("invalid email action type")
	}
}

func (a *API) sendEmailChangeEmails(r *http.Request, tx *storage.Connection, u *models.User, ottNew, ottCurrent *models.OneTimeToken, emailActionType, otp, otpNew string) error {
	mailer := a.Mailer()
	ctx := r.Context()
	config := a.config
	referrerURL := utilities.GetReferrer(r, config)
	externalURL := getExternalHost(ctx)
	if config.Hook.SendEmail.Enabled {
		emailData := mail.EmailData{
			Token:           otpNew,
			EmailActionType: emailActionType,
			RedirectTo:      referrerURL,
			SiteURL:         externalURL.String(),
			TokenHash:       ottNew.TokenHash,
		}
		if config.Mailer.SecureEmailChangeEnabled && u.GetEmail() != "" {
			emailData.TokenNew = otp
			emailData.TokenHashNew = ottCurrent.TokenHash
		}
		input := hooks.SendEmailInput{
			User:      u,
			EmailData: emailData,
		}
		output := hooks.SendEmailOutput{}
		return a.invokeHook(tx, r, &input, &output, config.Hook.SendEmail.URI)
	}

	wg := new(errgroup.Group)
	if config.Mailer.SecureEmailChangeEnabled && u.GetEmail() != "" {
		go mailer.EmailChangeMail(r, u, ottCurrent, otp, referrerURL, externalURL)
	}
	go mailer.EmailChangeMail(r, u, ottNew, otpNew, referrerURL, externalURL)
	if err := wg.Wait(); err != nil {
		// we return the first not-nil error observed
		return err
	}
	return nil
}
