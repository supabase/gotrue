package mailer

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/mailme"
	"gopkg.in/gomail.v2"
)

// Mailer defines the interface a mailer must implement.
type Mailer interface {
	InviteMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error
	ConfirmationMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error
	RecoveryMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error
	MagicLinkMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error
	EmailChangeMail(r *http.Request, user *models.User, ott *models.OneTimeToken, otp, referrerURL string, externalURL *url.URL) error
	ReauthenticateMail(r *http.Request, user *models.User, otp string) error
	ValidateEmail(email string) error
	GetEmailActionLink(ott *models.OneTimeToken, actionType, referrerURL string, externalURL *url.URL) (string, error)
}

type EmailParams struct {
	Token      string
	Type       string
	RedirectTo string
}

type EmailData struct {
	Token           string `json:"token"`
	TokenHash       string `json:"token_hash"`
	RedirectTo      string `json:"redirect_to"`
	EmailActionType string `json:"email_action_type"`
	SiteURL         string `json:"site_url"`
	TokenNew        string `json:"token_new"`
	TokenHashNew    string `json:"token_hash_new"`
}

// NewMailer returns a new gotrue mailer
func NewMailer(globalConfig *conf.GlobalConfiguration) Mailer {
	mail := gomail.NewMessage()

	mail.SetHeaders(map[string][]string{
		// Make the emails explicitly set to be HTML formatted (to cover older email clients)
		"Content-Type": {"text/html; charset=utf-8"},
		// so that messages are not grouped under each other
		"Message-ID": {fmt.Sprintf("<%s@gotrue-mailer>", uuid.Must(uuid.NewV4()).String())},
	})

	from := mail.FormatAddress(globalConfig.SMTP.AdminEmail, globalConfig.SMTP.SenderName)
	u, _ := url.ParseRequestURI(globalConfig.API.ExternalURL)

	var mailClient MailClient
	if globalConfig.SMTP.Host == "" {
		logrus.Infof("Noop mail client being used for %v", globalConfig.SiteURL)
		mailClient = &noopMailClient{}
	} else {
		mailClient = &mailme.Mailer{
			Host:      globalConfig.SMTP.Host,
			Port:      globalConfig.SMTP.Port,
			User:      globalConfig.SMTP.User,
			Pass:      globalConfig.SMTP.Pass,
			LocalName: u.Hostname(),
			From:      from,
			BaseURL:   globalConfig.SiteURL,
			Logger:    logrus.StandardLogger(),
		}
	}

	return &TemplateMailer{
		SiteURL: globalConfig.SiteURL,
		Config:  globalConfig,
		Mailer:  mailClient,
	}
}

func withDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getPath(filepath string, params *EmailParams) (*url.URL, error) {
	path := &url.URL{}
	if filepath != "" {
		if p, err := url.Parse(filepath); err != nil {
			return nil, err
		} else {
			path = p
		}
	}
	if params != nil {
		path.RawQuery = fmt.Sprintf("token=%s&type=%s&redirect_to=%s", url.QueryEscape(params.Token), url.QueryEscape(params.Type), encodeRedirectURL(params.RedirectTo))
	}
	return path, nil
}
