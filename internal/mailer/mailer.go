package mailer

import (
	"fmt"
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
	Send(user *models.User, subject, body string, data map[string]interface{}) error
	InviteMail(user *models.User, otp, referrerURL string, externalURL *url.URL) error
	ConfirmationMail(user *models.User, otp, referrerURL string, externalURL *url.URL) error
	RecoveryMail(user *models.User, otp, referrerURL string, externalURL *url.URL) error
	MagicLinkMail(user *models.User, otp, referrerURL string, externalURL *url.URL) error
	EmailChangeMail(user *models.User, otpNew, otpCurrent, referrerURL string, externalURL *url.URL) error
	ReauthenticateMail(user *models.User, otp string) error
	ValidateEmail(email string) error
	GetEmailActionLink(user *models.User, actionType, referrerURL string, externalURL *url.URL) (string, error)
}

type EmailParams struct {
	Token      string
	Type       string
	RedirectTo string
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

	var mailClient MailClient
	if globalConfig.SMTP.Host == "" {
		logrus.Infof("Noop mail client being used for %v", globalConfig.SiteURL)
		mailClient = &noopMailClient{}
	} else {
		mailClient = &mailme.Mailer{
			Host:    globalConfig.SMTP.Host,
			Port:    globalConfig.SMTP.Port,
			User:    globalConfig.SMTP.User,
			Pass:    globalConfig.SMTP.Pass,
			From:    from,
			BaseURL: globalConfig.SiteURL,
			Logger:  logrus.StandardLogger(),
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
