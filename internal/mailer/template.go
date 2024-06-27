package mailer

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/badoux/checkmail"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/models"
)

type MailClient interface {
	Mail(string, string, string, string, map[string]interface{}) error
}

// TemplateMailer will send mail and use templates from the site for easy mail styling
type TemplateMailer struct {
	SiteURL string
	Config  *conf.GlobalConfiguration
	Mailer  MailClient
}

func encodeRedirectURL(referrerURL string) string {
	if len(referrerURL) > 0 {
		if strings.ContainsAny(referrerURL, "&=#") {
			// if the string contains &, = or # it has not been URL
			// encoded by the caller, which means it should be URL
			// encoded by us otherwise, it should be taken as-is
			referrerURL = url.QueryEscape(referrerURL)
		}
	}
	return referrerURL
}

const (
	SignupVerification             = "signup"
	RecoveryVerification           = "recovery"
	InviteVerification             = "invite"
	MagicLinkVerification          = "magiclink"
	EmailChangeMailType            = "email_change"
	EmailOTPVerification           = "email"
	EmailChangeCurrentVerification = "email_change_current"
	EmailChangeNewVerification     = "email_change_new"
	ReauthenticationVerification   = "reauthentication"
)

const defaultInviteMail = `<h2>You have been invited</h2>

<p>You have been invited to create a user on {{ .SiteURL }}. Follow this link to accept the invite:</p>
<p><a href="{{ .ConfirmationURL }}">Accept the invite</a></p>
<p>Alternatively, enter the code: {{ .Token }}</p>`

const defaultConfirmationMail = `<h2>Confirm your email</h2>

<p>Follow this link to confirm your email:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your email address</a></p>
<p>Alternatively, enter the code: {{ .Token }}</p>
`

const defaultRecoveryMail = `<h2>Reset password</h2>

<p>Follow this link to reset the password for your user:</p>
<p><a href="{{ .ConfirmationURL }}">Reset password</a></p>
<p>Alternatively, enter the code: {{ .Token }}</p>`

const defaultMagicLinkMail = `<h2>Magic Link</h2>

<p>Follow this link to login:</p>
<p><a href="{{ .ConfirmationURL }}">Log In</a></p>
<p>Alternatively, enter the code: {{ .Token }}</p>`

const defaultEmailChangeMail = `<h2>Confirm email address change</h2>

<p>Follow this link to confirm the update of your email address from {{ .Email }} to {{ .NewEmail }}:</p>
<p><a href="{{ .ConfirmationURL }}">Change email address</a></p>
<p>Alternatively, enter the code: {{ .Token }}</p>`

const defaultReauthenticateMail = `<h2>Confirm reauthentication</h2>

<p>Enter the code: {{ .Token }}</p>`

// ValidateEmail returns nil if the email is valid,
// otherwise an error indicating the reason it is invalid
func (m TemplateMailer) ValidateEmail(email string) error {
	return checkmail.ValidateFormat(email)
}

// InviteMail sends a invite mail to a new user
func (m *TemplateMailer) InviteMail(r *http.Request, user *models.User, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Invite, &EmailParams{
		Token:      user.ConfirmationToken,
		Type:       InviteVerification,
		RedirectTo: referrerURL,
	})

	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.Email,
		"Token":           otp,
		"TokenHash":       user.ConfirmationToken,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
		"MailType":        InviteVerification,
	}

	return m.Mailer.Mail(
		user.GetEmail(),
		withDefault(m.Config.Mailer.Subjects.Invite, "You have been invited"),
		m.Config.Mailer.Templates.Invite,
		defaultInviteMail,
		data,
	)
}

// ConfirmationMail sends a signup confirmation mail to a new user
func (m *TemplateMailer) ConfirmationMail(r *http.Request, user *models.User, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Confirmation, &EmailParams{
		Token:      user.ConfirmationToken,
		Type:       SignupVerification,
		RedirectTo: referrerURL,
	})
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.Email,
		"Token":           otp,
		"TokenHash":       user.ConfirmationToken,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
		"MailType":        SignupVerification,
	}

	return m.Mailer.Mail(
		user.GetEmail(),
		withDefault(m.Config.Mailer.Subjects.Confirmation, "Confirm Your Email"),
		m.Config.Mailer.Templates.Confirmation,
		defaultConfirmationMail,
		data,
	)
}

// ReauthenticateMail sends a reauthentication mail to an authenticated user
func (m *TemplateMailer) ReauthenticateMail(r *http.Request, user *models.User, otp string) error {
	data := map[string]interface{}{
		"SiteURL": m.Config.SiteURL,
		"Email":   user.Email,
		"Token":   otp,
		"Data":    user.UserMetaData,
	}

	return m.Mailer.Mail(
		user.GetEmail(),
		withDefault(m.Config.Mailer.Subjects.Reauthentication, "Confirm reauthentication"),
		m.Config.Mailer.Templates.Reauthentication,
		defaultReauthenticateMail,
		data,
	)
}

// EmailChangeMail sends an email change confirmation mail to a user
func (m *TemplateMailer) EmailChangeMail(r *http.Request, user *models.User, otpNew, otpCurrent, referrerURL string, externalURL *url.URL) error {
	type Email struct {
		Address   string
		Otp       string
		TokenHash string
		Subject   string
		Template  string
	}
	emails := []Email{
		{
			Address:   user.EmailChange,
			Otp:       otpNew,
			TokenHash: user.EmailChangeTokenNew,
			Subject:   withDefault(m.Config.Mailer.Subjects.EmailChange, "Confirm Email Change"),
			Template:  m.Config.Mailer.Templates.EmailChange,
		},
	}

	currentEmail := user.GetEmail()
	if m.Config.Mailer.SecureEmailChangeEnabled && currentEmail != "" {
		emails = append(emails, Email{
			Address:   currentEmail,
			Otp:       otpCurrent,
			TokenHash: user.EmailChangeTokenCurrent,
			Subject:   withDefault(m.Config.Mailer.Subjects.Confirmation, "Confirm Email Address"),
			Template:  m.Config.Mailer.Templates.EmailChange,
		})
	}

	errors := make(chan error)
	for _, email := range emails {
		path, err := getPath(
			m.Config.Mailer.URLPaths.EmailChange,
			&EmailParams{
				Token:      email.TokenHash,
				Type:       EmailChangeMailType,
				RedirectTo: referrerURL,
			},
		)
		if err != nil {
			return err
		}
		go func(address, token, tokenHash, template string) {
			data := map[string]interface{}{
				"SiteURL":         m.Config.SiteURL,
				"ConfirmationURL": externalURL.ResolveReference(path).String(),
				"Email":           user.GetEmail(),
				"NewEmail":        user.EmailChange,
				"Token":           token,
				"TokenHash":       tokenHash,
				"SendingTo":       address,
				"Data":            user.UserMetaData,
				"RedirectTo":      referrerURL,
				"MailType":        EmailChangeMailType,
			}
			errors <- m.Mailer.Mail(
				address,
				withDefault(m.Config.Mailer.Subjects.EmailChange, "Confirm Email Change"),
				template,
				defaultEmailChangeMail,
				data,
			)
		}(email.Address, email.Otp, email.TokenHash, email.Template)
	}

	for i := 0; i < len(emails); i++ {
		e := <-errors
		if e != nil {
			return e
		}
	}

	return nil
}

// RecoveryMail sends a password recovery mail
func (m *TemplateMailer) RecoveryMail(r *http.Request, user *models.User, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
		Token:      user.RecoveryToken,
		Type:       RecoveryVerification,
		RedirectTo: referrerURL,
	})
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.Email,
		"Token":           otp,
		"TokenHash":       user.RecoveryToken,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
		"MailType":        RecoveryVerification,
	}

	return m.Mailer.Mail(
		user.GetEmail(),
		withDefault(m.Config.Mailer.Subjects.Recovery, "Reset Your Password"),
		m.Config.Mailer.Templates.Recovery,
		defaultRecoveryMail,
		data,
	)
}

// MagicLinkMail sends a login link mail
func (m *TemplateMailer) MagicLinkMail(r *http.Request, user *models.User, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
		Token:      user.RecoveryToken,
		Type:       MagicLinkVerification,
		RedirectTo: referrerURL,
	})
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.Email,
		"Token":           otp,
		"TokenHash":       user.RecoveryToken,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
		"MailType":        MagicLinkVerification,
	}

	return m.Mailer.Mail(
		user.GetEmail(),
		withDefault(m.Config.Mailer.Subjects.MagicLink, "Your Magic Link"),
		m.Config.Mailer.Templates.MagicLink,
		defaultMagicLinkMail,
		data,
	)
}

// Send can be used to send one-off emails to users
func (m TemplateMailer) Send(user *models.User, subject, body string, data map[string]interface{}) error {
	return m.Mailer.Mail(
		user.GetEmail(),
		subject,
		"",
		body,
		data,
	)
}

// GetEmailActionLink returns a magiclink, recovery or invite link based on the actionType passed.
func (m TemplateMailer) GetEmailActionLink(user *models.User, actionType, referrerURL string, externalURL *url.URL) (string, error) {
	var err error
	var path *url.URL

	switch actionType {
	case MagicLinkVerification:
		path, err = getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
			Token:      user.RecoveryToken,
			Type:       MagicLinkVerification,
			RedirectTo: referrerURL,
		})
	case RecoveryVerification:
		path, err = getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
			Token:      user.RecoveryToken,
			Type:       RecoveryVerification,
			RedirectTo: referrerURL,
		})
	case InviteVerification:
		path, err = getPath(m.Config.Mailer.URLPaths.Invite, &EmailParams{
			Token:      user.ConfirmationToken,
			Type:       InviteVerification,
			RedirectTo: referrerURL,
		})
	case SignupVerification:
		path, err = getPath(m.Config.Mailer.URLPaths.Confirmation, &EmailParams{
			Token:      user.ConfirmationToken,
			Type:       SignupVerification,
			RedirectTo: referrerURL,
		})
	case "email_change_current":
		path, err = getPath(m.Config.Mailer.URLPaths.EmailChange, &EmailParams{
			Token:      user.EmailChangeTokenCurrent,
			Type:       "email_change",
			RedirectTo: referrerURL,
		})
	case "email_change_new":
		path, err = getPath(m.Config.Mailer.URLPaths.EmailChange, &EmailParams{
			Token:      user.EmailChangeTokenNew,
			Type:       "email_change",
			RedirectTo: referrerURL,
		})
	default:
		return "", fmt.Errorf("invalid email action link type: %s", actionType)
	}
	if err != nil {
		return "", err
	}
	return externalURL.ResolveReference(path).String(), nil
}
