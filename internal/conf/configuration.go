package conf

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gobwas/glob"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

const defaultMinPasswordLength int = 6
const defaultChallengeExpiryDuration float64 = 300
const defaultFlowStateExpiryDuration time.Duration = 300 * time.Second

// Time is used to represent timestamps in the configuration, as envconfig has
// trouble parsing empty strings, due to time.Time.UnmarshalText().
type Time struct {
	time.Time
}

func (t *Time) UnmarshalText(text []byte) error {
	trimed := bytes.TrimSpace(text)

	if len(trimed) < 1 {
		t.Time = time.Time{}
	} else {
		if err := t.Time.UnmarshalText(trimed); err != nil {
			return err
		}
	}

	return nil
}

// OAuthProviderConfiguration holds all config related to external account providers.
type OAuthProviderConfiguration struct {
	ClientID    []string `json:"client_id" split_words:"true"`
	Secret      string   `json:"secret"`
	RedirectURI string   `json:"redirect_uri" split_words:"true"`
	URL         string   `json:"url"`
	ApiURL      string   `json:"api_url" split_words:"true"`
	Enabled     bool     `json:"enabled"`
}

type EmailProviderConfiguration struct {
	Enabled bool `json:"enabled" default:"true"`
}

// DBConfiguration holds all the database related configuration.
type DBConfiguration struct {
	Driver    string `json:"driver" required:"true"`
	URL       string `json:"url" envconfig:"DATABASE_URL" required:"true"`
	Namespace string `json:"namespace" envconfig:"DB_NAMESPACE" default:"auth"`
	// MaxPoolSize defaults to 0 (unlimited).
	MaxPoolSize       int           `json:"max_pool_size" split_words:"true"`
	MaxIdlePoolSize   int           `json:"max_idle_pool_size" split_words:"true"`
	ConnMaxLifetime   time.Duration `json:"conn_max_lifetime,omitempty" split_words:"true"`
	ConnMaxIdleTime   time.Duration `json:"conn_max_idle_time,omitempty" split_words:"true"`
	HealthCheckPeriod time.Duration `json:"health_check_period" split_words:"true"`
	MigrationsPath    string        `json:"migrations_path" split_words:"true" default:"./migrations"`
	CleanupEnabled    bool          `json:"cleanup_enabled" split_words:"true" default:"false"`
}

func (c *DBConfiguration) Validate() error {
	return nil
}

// JWTConfiguration holds all the JWT related configuration.
type JWTConfiguration struct {
	Secret           string   `json:"secret" required:"true"`
	Exp              int      `json:"exp"`
	Aud              string   `json:"aud"`
	AdminGroupName   string   `json:"admin_group_name" split_words:"true"`
	AdminRoles       []string `json:"admin_roles" split_words:"true"`
	DefaultGroupName string   `json:"default_group_name" split_words:"true"`
	Issuer           string   `json:"issuer"`
	KeyID            string   `json:"key_id" split_words:"true"`
}

// MFAConfiguration holds all the MFA related Configuration
type MFAConfiguration struct {
	Enabled                     bool    `default:"false"`
	ChallengeExpiryDuration     float64 `json:"challenge_expiry_duration" default:"300" split_words:"true"`
	RateLimitChallengeAndVerify float64 `split_words:"true" default:"15"`
	MaxEnrolledFactors          float64 `split_words:"true" default:"10"`
	MaxVerifiedFactors          int     `split_words:"true" default:"10"`
}

type APIConfiguration struct {
	Host            string
	Port            string `envconfig:"PORT" default:"8081"`
	Endpoint        string
	RequestIDHeader string `envconfig:"REQUEST_ID_HEADER"`
	ExternalURL     string `json:"external_url" envconfig:"API_EXTERNAL_URL" required:"true"`
}

func (a *APIConfiguration) Validate() error {
	_, err := url.ParseRequestURI(a.ExternalURL)
	if err != nil {
		return err
	}

	return nil
}

// GlobalConfiguration holds all the configuration that applies to all instances.
type GlobalConfiguration struct {
	API                   APIConfiguration
	DB                    DBConfiguration
	External              ProviderConfiguration
	Logging               LoggingConfig  `envconfig:"LOG"`
	Profiler              ProfilerConfig `envconfig:"PROFILER"`
	OperatorToken         string         `split_words:"true" required:"false"`
	Tracing               TracingConfig
	Metrics               MetricsConfig
	SMTP                  SMTPConfiguration
	RateLimitHeader       string  `split_words:"true"`
	RateLimitEmailSent    float64 `split_words:"true" default:"30"`
	RateLimitSmsSent      float64 `split_words:"true" default:"30"`
	RateLimitVerify       float64 `split_words:"true" default:"30"`
	RateLimitTokenRefresh float64 `split_words:"true" default:"30"`
	RateLimitSso          float64 `split_words:"true" default:"30"`

	SiteURL           string   `json:"site_url" split_words:"true" required:"true"`
	URIAllowList      []string `json:"uri_allow_list" split_words:"true"`
	URIAllowListMap   map[string]glob.Glob
	PasswordMinLength int                      `json:"password_min_length" split_words:"true"`
	JWT               JWTConfiguration         `json:"jwt"`
	Mailer            MailerConfiguration      `json:"mailer"`
	Sms               SmsProviderConfiguration `json:"sms"`
	DisableSignup     bool                     `json:"disable_signup" split_words:"true"`
	Webhook           WebhookConfig            `json:"webhook" split_words:"true"`
	Security          SecurityConfiguration    `json:"security"`
	MFA               MFAConfiguration         `json:"MFA"`
	Cookie            struct {
		Key      string `json:"key"`
		Domain   string `json:"domain"`
		Duration int    `json:"duration"`
	} `json:"cookies"`
	SAML SAMLConfiguration `json:"saml"`
	CORS CORSConfiguration `json:"cors"`
}

type CORSConfiguration struct {
	AllowedHeaders []string `json:"allowed_headers" split_words:"true"`
}

func (c *CORSConfiguration) AllAllowedHeaders(defaults []string) []string {
	set := make(map[string]bool)
	for _, header := range defaults {
		set[header] = true
	}

	var result []string
	result = append(result, defaults...)

	for _, header := range c.AllowedHeaders {
		if !set[header] {
			result = append(result, header)
		}

		set[header] = true
	}

	return result
}

// EmailContentConfiguration holds the configuration for emails, both subjects and template URLs.
type EmailContentConfiguration struct {
	Invite           string `json:"invite"`
	Confirmation     string `json:"confirmation"`
	Recovery         string `json:"recovery"`
	EmailChange      string `json:"email_change" split_words:"true"`
	MagicLink        string `json:"magic_link" split_words:"true"`
	Reauthentication string `json:"reauthentication"`
}

type ProviderConfiguration struct {
	Apple                   OAuthProviderConfiguration `json:"apple"`
	Azure                   OAuthProviderConfiguration `json:"azure"`
	Bitbucket               OAuthProviderConfiguration `json:"bitbucket"`
	Discord                 OAuthProviderConfiguration `json:"discord"`
	Facebook                OAuthProviderConfiguration `json:"facebook"`
	Figma                   OAuthProviderConfiguration `json:"figma"`
	Github                  OAuthProviderConfiguration `json:"github"`
	Gitlab                  OAuthProviderConfiguration `json:"gitlab"`
	Google                  OAuthProviderConfiguration `json:"google"`
	Kakao                   OAuthProviderConfiguration `json:"kakao"`
	Naver                   OAuthProviderConfiguration `json:"naver"`
	Notion                  OAuthProviderConfiguration `json:"notion"`
	Keycloak                OAuthProviderConfiguration `json:"keycloak"`
	Linkedin                OAuthProviderConfiguration `json:"linkedin"`
	Spotify                 OAuthProviderConfiguration `json:"spotify"`
	Slack                   OAuthProviderConfiguration `json:"slack"`
	Twitter                 OAuthProviderConfiguration `json:"twitter"`
	Twitch                  OAuthProviderConfiguration `json:"twitch"`
	WorkOS                  OAuthProviderConfiguration `json:"workos"`
	Email                   EmailProviderConfiguration `json:"email"`
	Phone                   PhoneProviderConfiguration `json:"phone"`
	Zoom                    OAuthProviderConfiguration `json:"zoom"`
	IosBundleId             string                     `json:"ios_bundle_id" split_words:"true"`
	RedirectURL             string                     `json:"redirect_url"`
	AllowedIdTokenIssuers   []string                   `json:"allowed_id_token_issuers" split_words:"true"`
	FlowStateExpiryDuration time.Duration              `json:"flow_state_expiry_duration" split_words:"true"`
}

type SMTPConfiguration struct {
	MaxFrequency time.Duration `json:"max_frequency" split_words:"true"`
	Host         string        `json:"host"`
	Port         int           `json:"port,omitempty" default:"587"`
	User         string        `json:"user"`
	Pass         string        `json:"pass,omitempty"`
	AdminEmail   string        `json:"admin_email" split_words:"true"`
	SenderName   string        `json:"sender_name" split_words:"true"`
}

func (c *SMTPConfiguration) Validate() error {
	return nil
}

type MailerConfiguration struct {
	Autoconfirm              bool                      `json:"autoconfirm"`
	Subjects                 EmailContentConfiguration `json:"subjects"`
	Templates                EmailContentConfiguration `json:"templates"`
	URLPaths                 EmailContentConfiguration `json:"url_paths"`
	SecureEmailChangeEnabled bool                      `json:"secure_email_change_enabled" split_words:"true" default:"true"`
	OtpExp                   uint                      `json:"otp_exp" split_words:"true"`
	OtpLength                int                       `json:"otp_length" split_words:"true"`
}

type PhoneProviderConfiguration struct {
	Enabled bool `json:"enabled" default:"false"`
}

type SmsProviderConfiguration struct {
	Autoconfirm       bool               `json:"autoconfirm"`
	MaxFrequency      time.Duration      `json:"max_frequency" split_words:"true"`
	OtpExp            uint               `json:"otp_exp" split_words:"true"`
	OtpLength         int                `json:"otp_length" split_words:"true"`
	Provider          string             `json:"provider"`
	Template          string             `json:"template"`
	TestOTP           map[string]string  `json:"test_otp" split_words:"true"`
	TestOTPValidUntil Time               `json:"test_otp_valid_until" split_words:"true"`
	SMSTemplate       *template.Template `json:"-"`

	Twilio       TwilioProviderConfiguration       `json:"twilio"`
	TwilioVerify TwilioVerifyProviderConfiguration `json:"twilio_verify" split_words:"true"`
	Messagebird  MessagebirdProviderConfiguration  `json:"messagebird"`
	Textlocal    TextlocalProviderConfiguration    `json:"textlocal"`
	Vonage       VonageProviderConfiguration       `json:"vonage"`
}

func (c *SmsProviderConfiguration) GetTestOTP(phone string, now time.Time) (string, bool) {
	if c.TestOTP != nil && (c.TestOTPValidUntil.Time.IsZero() || now.Before(c.TestOTPValidUntil.Time)) {
		testOTP, ok := c.TestOTP[phone]
		return testOTP, ok
	}

	return "", false
}

type TwilioProviderConfiguration struct {
	AccountSid        string `json:"account_sid" split_words:"true"`
	AuthToken         string `json:"auth_token" split_words:"true"`
	MessageServiceSid string `json:"message_service_sid" split_words:"true"`
}

type TwilioVerifyProviderConfiguration struct {
	AccountSid        string `json:"account_sid" split_words:"true"`
	AuthToken         string `json:"auth_token" split_words:"true"`
	MessageServiceSid string `json:"message_service_sid" split_words:"true"`
}

type MessagebirdProviderConfiguration struct {
	AccessKey  string `json:"access_key" split_words:"true"`
	Originator string `json:"originator" split_words:"true"`
}

type TextlocalProviderConfiguration struct {
	ApiKey string `json:"api_key" split_words:"true"`
	Sender string `json:"sender" split_words:"true"`
}

type VonageProviderConfiguration struct {
	ApiKey    string `json:"api_key" split_words:"true"`
	ApiSecret string `json:"api_secret" split_words:"true"`
	From      string `json:"from" split_words:"true"`
}

type CaptchaConfiguration struct {
	Enabled  bool   `json:"enabled" default:"false"`
	Provider string `json:"provider" default:"hcaptcha"`
	Secret   string `json:"provider_secret"`
}

func (c *CaptchaConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Provider != "hcaptcha" && c.Provider != "turnstile" {
		return fmt.Errorf("unsupported captcha provider: %s", c.Provider)
	}

	c.Secret = strings.TrimSpace(c.Secret)

	if c.Secret == "" {
		return errors.New("captcha provider secret is empty")
	}

	return nil
}

type SecurityConfiguration struct {
	Captcha                               CaptchaConfiguration `json:"captcha"`
	RefreshTokenRotationEnabled           bool                 `json:"refresh_token_rotation_enabled" split_words:"true" default:"true"`
	RefreshTokenReuseInterval             int                  `json:"refresh_token_reuse_interval" split_words:"true"`
	UpdatePasswordRequireReauthentication bool                 `json:"update_password_require_reauthentication" split_words:"true"`
}

func (c *SecurityConfiguration) Validate() error {
	return c.Captcha.Validate()
}

func loadEnvironment(filename string) error {
	var err error
	if filename != "" {
		err = godotenv.Overload(filename)
	} else {
		err = godotenv.Load()
		// handle if .env file does not exist, this is OK
		if os.IsNotExist(err) {
			return nil
		}
	}
	return err
}

type WebhookConfig struct {
	URL        string   `json:"url"`
	Retries    int      `json:"retries"`
	TimeoutSec int      `json:"timeout_sec"`
	Secret     string   `json:"secret"`
	Events     []string `json:"events"`
}

func (w *WebhookConfig) HasEvent(event string) bool {
	for _, name := range w.Events {
		if event == name {
			return true
		}
	}
	return false
}

func LoadGlobal(filename string) (*GlobalConfiguration, error) {
	if err := loadEnvironment(filename); err != nil {
		return nil, err
	}

	config := new(GlobalConfiguration)
	if err := envconfig.Process("gotrue", config); err != nil {
		return nil, err
	}

	if err := config.ApplyDefaults(); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.SAML.Enabled {
		if err := config.SAML.PopulateFields(config.API.ExternalURL); err != nil {
			return nil, err
		}
	} else {
		config.SAML.PrivateKey = ""
	}
	if config.Sms.Provider != "" {
		SMSTemplate := config.Sms.Template
		if SMSTemplate == "" {
			SMSTemplate = "Your code is {{ .Code }}"
		}
		template, err := template.New("").Parse(SMSTemplate)
		if err != nil {
			return nil, err
		}
		config.Sms.SMSTemplate = template
	}

	return config, nil
}

// ApplyDefaults sets defaults for a GlobalConfiguration
func (config *GlobalConfiguration) ApplyDefaults() error {
	if config.JWT.AdminGroupName == "" {
		config.JWT.AdminGroupName = "admin"
	}

	if config.JWT.AdminRoles == nil || len(config.JWT.AdminRoles) == 0 {
		config.JWT.AdminRoles = []string{"service_role", "supabase_admin"}
	}

	if config.JWT.Exp == 0 {
		config.JWT.Exp = 3600
	}

	if config.Mailer.URLPaths.Invite == "" {
		config.Mailer.URLPaths.Invite = "/verify"
	}

	if config.Mailer.URLPaths.Confirmation == "" {
		config.Mailer.URLPaths.Confirmation = "/verify"
	}

	if config.Mailer.URLPaths.Recovery == "" {
		config.Mailer.URLPaths.Recovery = "/verify"
	}

	if config.Mailer.URLPaths.EmailChange == "" {
		config.Mailer.URLPaths.EmailChange = "/verify"
	}

	if config.Mailer.OtpExp == 0 {
		config.Mailer.OtpExp = 86400 // 1 day
	}

	if config.Mailer.OtpLength == 0 || config.Mailer.OtpLength < 6 || config.Mailer.OtpLength > 10 {
		// 6-digit otp by default
		config.Mailer.OtpLength = 6
	}

	if config.SMTP.MaxFrequency == 0 {
		config.SMTP.MaxFrequency = 1 * time.Minute
	}

	if config.Sms.MaxFrequency == 0 {
		config.Sms.MaxFrequency = 1 * time.Minute
	}

	if config.Sms.OtpExp == 0 {
		config.Sms.OtpExp = 60
	}

	if config.Sms.OtpLength == 0 || config.Sms.OtpLength < 6 || config.Sms.OtpLength > 10 {
		// 6-digit otp by default
		config.Sms.OtpLength = 6
	}

	if len(config.Sms.Template) == 0 {
		config.Sms.Template = ""
	}

	if config.Cookie.Key == "" {
		config.Cookie.Key = "sb"
	}

	if config.Cookie.Domain == "" {
		config.Cookie.Domain = ""
	}

	if config.Cookie.Duration == 0 {
		config.Cookie.Duration = 86400
	}

	if config.URIAllowList == nil {
		config.URIAllowList = []string{}
	}

	if config.URIAllowList != nil {
		config.URIAllowListMap = make(map[string]glob.Glob)
		for _, uri := range config.URIAllowList {
			g := glob.MustCompile(uri, '.', '/')
			config.URIAllowListMap[uri] = g
		}
	}

	if config.PasswordMinLength < defaultMinPasswordLength {
		config.PasswordMinLength = defaultMinPasswordLength
	}
	if config.MFA.ChallengeExpiryDuration < defaultChallengeExpiryDuration {
		config.MFA.ChallengeExpiryDuration = defaultChallengeExpiryDuration
	}
	if config.External.FlowStateExpiryDuration < defaultFlowStateExpiryDuration {
		config.External.FlowStateExpiryDuration = defaultFlowStateExpiryDuration
	}

	if len(config.External.AllowedIdTokenIssuers) == 0 {
		config.External.AllowedIdTokenIssuers = append(config.External.AllowedIdTokenIssuers, "https://appleid.apple.com", "https://accounts.google.com")
	}

	return nil
}

// Validate validates all of configuration.
func (c *GlobalConfiguration) Validate() error {
	validatables := []interface {
		Validate() error
	}{
		&c.API,
		&c.DB,
		&c.Tracing,
		&c.Metrics,
		&c.SMTP,
		&c.SAML,
		&c.Security,
	}

	for _, validatable := range validatables {
		if err := validatable.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (o *OAuthProviderConfiguration) ValidateOAuth() error {
	if !o.Enabled {
		return errors.New("provider is not enabled")
	}
	if len(o.ClientID) == 0 {
		return errors.New("missing OAuth client ID")
	}
	if o.Secret == "" {
		return errors.New("missing OAuth secret")
	}
	if o.RedirectURI == "" {
		return errors.New("missing redirect URI")
	}
	return nil
}

func (t *TwilioProviderConfiguration) Validate() error {
	if t.AccountSid == "" {
		return errors.New("missing Twilio account SID")
	}
	if t.AuthToken == "" {
		return errors.New("missing Twilio auth token")
	}
	if t.MessageServiceSid == "" {
		return errors.New("missing Twilio message service SID or Twilio phone number")
	}
	return nil
}

func (t *TwilioVerifyProviderConfiguration) Validate() error {
	if t.AccountSid == "" {
		return errors.New("missing Twilio account SID")
	}
	if t.AuthToken == "" {
		return errors.New("missing Twilio auth token")
	}
	if t.MessageServiceSid == "" {
		return errors.New("missing Twilio message service SID or Twilio phone number")
	}
	return nil
}

func (t *MessagebirdProviderConfiguration) Validate() error {
	if t.AccessKey == "" {
		return errors.New("missing Messagebird access key")
	}
	if t.Originator == "" {
		return errors.New("missing Messagebird originator")
	}
	return nil
}

func (t *TextlocalProviderConfiguration) Validate() error {
	if t.ApiKey == "" {
		return errors.New("missing Textlocal API key")
	}
	if t.Sender == "" {
		return errors.New("missing Textlocal sender")
	}
	return nil
}

func (t *VonageProviderConfiguration) Validate() error {
	if t.ApiKey == "" {
		return errors.New("missing Vonage API key")
	}
	if t.ApiSecret == "" {
		return errors.New("missing Vonage API secret")
	}
	if t.From == "" {
		return errors.New("missing Vonage 'from' parameter")
	}
	return nil
}

func (t *SmsProviderConfiguration) IsTwilioVerifyProvider() bool {
	return t.Provider == "twilio_verify"
}
