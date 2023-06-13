package sms_provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supabase/gotrue/internal/conf"
	"github.com/supabase/gotrue/internal/utilities"
)

const (
	verifyServiceApiBase = "https://verify.twilio.com/v2/Services/"
)

type TwilioVerifyProvider struct {
	Config  *conf.TwilioProviderConfiguration
	APIPath string
}

type VerificationResponse struct {
	To           string `json:"to"`
	Status       string `json:"status"`
	Channel      string `json:"channel"`
	Valid        bool   `json:"valid"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// See: https://www.twilio.com/docs/verify/api/verification-check
type VerificationCheckResponse struct {
	To           string `json:"to"`
	Status       string `json:"status"`
	Channel      string `json:"channel"`
	Valid        bool   `json:"valid"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// Creates a SmsProvider with the Twilio Config
func NewTwilioVerifyProvider(config conf.TwilioProviderConfiguration) (SmsProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	var apiPath string
	if config.VerifyEnabled {
		apiPath = verifyServiceApiBase + config.MessageServiceSid + "/Verifications"
	} else {
		apiPath = defaultTwilioApiBase + "/" + apiVersion + "/" + "Accounts" + "/" + config.AccountSid + "/Messages.json"
	}

	return &TwilioVerifyProvider{
		Config:  &config,
		APIPath: apiPath,
	}, nil
}

func (t *TwilioVerifyProvider) SendMessage(phone string, message string, channel string) error {
	switch channel {
	case SMSProvider, WhatsappProvider:
		return t.SendSms(phone, message, channel)
	default:
		return fmt.Errorf("channel type %q is not supported for Twilio", channel)
	}
}

// Send an SMS containing the OTP with Twilio's API
func (t *TwilioVerifyProvider) SendSms(phone, message, channel string) error {

	// Unlike Programmable Messaging, Verify does not require a prefix for channel
	// E164 format is also guaranteed by the time this function is called
	body := url.Values{
		"To":      {phone},
		"Channel": {channel},
	}
	client := &http.Client{Timeout: defaultTimeout}
	r, err := http.NewRequest("POST", t.APIPath, strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(t.Config.AccountSid, t.Config.AuthToken)
	res, err := client.Do(r)
	defer utilities.SafeClose(res.Body)
	if err != nil {
		return err
	}
	if !(res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated) {
		resp := &twilioErrResponse{}
		if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
			return err
		}
		return resp
	}
	return nil
}

func (t *TwilioVerifyProvider) VerifyOTP(phone, code string) error {
	// Additional guard check
	if !t.Config.VerifyEnabled {
		return fmt.Errorf("twilio verify is not enabled")
	}
	verifyPath := verifyServiceApiBase + t.Config.MessageServiceSid + "/VerificationCheck"

	body := url.Values{
		"To":   {phone}, // twilio api requires "+" extension to be included
		"Code": {code},
	}
	client := &http.Client{Timeout: defaultTimeout}
	r, err := http.NewRequest("POST", verifyPath, strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(t.Config.AccountSid, t.Config.AuthToken)
	res, err := client.Do(r)
	defer utilities.SafeClose(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		resp := &twilioErrResponse{}
		if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
			return err
		}
		return resp
	}
	resp := &VerificationCheckResponse{}
	derr := json.NewDecoder(res.Body).Decode(resp)
	if derr != nil {
		return derr
	}

	if resp.Status != "approved" || !resp.Valid {
		return fmt.Errorf("twilio verification error: %v %v", resp.ErrorMessage, resp.Status)
	}

	return nil
}
