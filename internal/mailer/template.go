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
	EmailChangeVerification        = "email_change"
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
func (m *TemplateMailer) InviteMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Invite, &EmailParams{
		Token:      ott.TokenHash,
		Type:       "invite",
		RedirectTo: referrerURL,
	})

	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.GetEmail(),
		"Token":           otp,
		"TokenHash":       ott.TokenHash,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
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
func (m *TemplateMailer) ConfirmationMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Confirmation, &EmailParams{
		Token:      ott.TokenHash,
		Type:       "signup",
		RedirectTo: referrerURL,
	})
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.GetEmail(),
		"Token":           otp,
		"TokenHash":       ott.TokenHash,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
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
		"Email":   user.GetEmail(),
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
func (m *TemplateMailer) EmailChangeMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(
		m.Config.Mailer.URLPaths.EmailChange,
		&EmailParams{
			Token:      ott.TokenHash,
			Type:       "email_change",
			RedirectTo: referrerURL,
		},
	)
	if err != nil {
		return err
	}

	sendingTo := user.EmailChange
	if ott.TokenType == models.EmailChangeTokenCurrent {
		sendingTo = user.GetEmail()
	}

	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.GetEmail(),
		"NewEmail":        user.EmailChange,
		"Token":           otp,
		"TokenHash":       ott.TokenHash,
		"SendingTo":       sendingTo,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
	}
	return m.Mailer.Mail(
		sendingTo,
		withDefault(m.Config.Mailer.Subjects.EmailChange, "Confirm Email Change"),
		m.Config.Mailer.Templates.EmailChange,
		defaultEmailChangeMail,
		data,
	)
}

// RecoveryMail sends a password recovery mail
func (m *TemplateMailer) RecoveryMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
		Token:      ott.TokenHash,
		Type:       "recovery",
		RedirectTo: referrerURL,
	})
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.GetEmail(),
		"Token":           otp,
		"TokenHash":       ott.TokenHash,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
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
func (m *TemplateMailer) MagicLinkMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error {
	path, err := getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
		Token:      ott.TokenHash,
		Type:       "magiclink",
		RedirectTo: referrerURL,
	})
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": externalURL.ResolveReference(path).String(),
		"Email":           user.GetEmail(),
		"Token":           otp,
		"TokenHash":       ott.TokenHash,
		"Data":            user.UserMetaData,
		"RedirectTo":      referrerURL,
	}

	return m.Mailer.Mail(
		user.GetEmail(),
		withDefault(m.Config.Mailer.Subjects.MagicLink, "Your Magic Link"),
		m.Config.Mailer.Templates.MagicLink,
		defaultMagicLinkMail,
		data,
	)
}

// GetEmailActionLink returns a magiclink, recovery or invite link based on the actionType passed.
func (m TemplateMailer) GetEmailActionLink(ott *models.OneTimeToken, actionType, referrerURL string, externalURL *url.URL) (string, error) {
	if ott == nil {
		return "", fmt.Errorf("ott is nil")
	}

	var err error
	var path *url.URL

	switch actionType {
	case "magiclink":
		path, err = getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
			Token:      ott.TokenHash,
			Type:       "magiclink",
			RedirectTo: referrerURL,
		})
	case "recovery":
		path, err = getPath(m.Config.Mailer.URLPaths.Recovery, &EmailParams{
			Token:      ott.TokenHash,
			Type:       "recovery",
			RedirectTo: referrerURL,
		})
	case "invite":
		path, err = getPath(m.Config.Mailer.URLPaths.Invite, &EmailParams{
			Token:      ott.TokenHash,
			Type:       "invite",
			RedirectTo: referrerURL,
		})
	case "signup":
		path, err = getPath(m.Config.Mailer.URLPaths.Confirmation, &EmailParams{
			Token:      ott.TokenHash,
			Type:       "signup",
			RedirectTo: referrerURL,
		})
	case "email_change_current":
		path, err = getPath(m.Config.Mailer.URLPaths.EmailChange, &EmailParams{
			Token:      ott.TokenHash,
			Type:       "email_change",
			RedirectTo: referrerURL,
		})
	case "email_change_new":
		path, err = getPath(m.Config.Mailer.URLPaths.EmailChange, &EmailParams{
			Token:      ott.TokenHash,
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
