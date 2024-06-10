package api

import (
	"bytes"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/supabase/auth/internal/hooks"

	"github.com/pkg/errors"
	"github.com/supabase/auth/internal/api/sms_provider"
	"github.com/supabase/auth/internal/crypto"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/storage"
)

var e164Format = regexp.MustCompile("^[1-9][0-9]{1,14}$")

const (
	phoneConfirmationOtp     = "confirmation"
	phoneReauthenticationOtp = "reauthentication"
)

func validatePhone(phone string) (string, error) {
	phone = formatPhoneNumber(phone)
	if isValid := validateE164Format(phone); !isValid {
		return "", badRequestError(ErrorCodeValidationFailed, "Invalid phone number format (E.164 required)")
	}
	return phone, nil
}

// validateE164Format checks if phone number follows the E.164 format
func validateE164Format(phone string) bool {
	return e164Format.MatchString(phone)
}

// formatPhoneNumber removes "+" and whitespaces in a phone number
func formatPhoneNumber(phone string) string {
	return strings.ReplaceAll(strings.TrimPrefix(phone, "+"), " ", "")
}

// sendPhoneConfirmation sends an otp to the user's phone number
func (a *API) sendPhoneConfirmation(r *http.Request, tx *storage.Connection, user *models.User, phone, otpType string, smsProvider sms_provider.SmsProvider, channel string) (string, error) {
	config := a.config
	var sentAt *time.Time

	includeFields := []string{}
	switch otpType {
	case phoneChangeVerification:
		sentAt = user.PhoneChangeSentAt
		user.PhoneChange = phone
		includeFields = append(includeFields, "phone_change", "phone_change_sent_at")
	case phoneConfirmationOtp:
		sentAt = user.ConfirmationSentAt
		includeFields = append(includeFields, "confirmation_sent_at")
	case phoneReauthenticationOtp:
		sentAt = user.ReauthenticationSentAt
		includeFields = append(includeFields, "reauthentication_sent_at")
	default:
		return "", internalServerError("invalid otp type")
	}

	// intentionally keeping this before the test OTP, so that the behavior
	// of regular and test OTPs is similar
	if sentAt != nil && !sentAt.Add(config.Sms.MaxFrequency).Before(time.Now()) {
		return "", MaxFrequencyLimitError
	}

	now := time.Now()

	var otp, messageID string
	var err error

	if testOTP, ok := config.Sms.GetTestOTP(phone, now); ok {
		otp = testOTP
		messageID = "test-otp"
	}

	if otp == "" { // not using test OTPs
		otp, err = crypto.GenerateOtp(config.Sms.OtpLength)
		if err != nil {
			return "", internalServerError("error generating otp").WithInternalError(err)
		}

		message, err := generateSMSFromTemplate(config.Sms.SMSTemplate, otp)
		if err != nil {
			return "", err
		}

		// Hook should only be called if SMS autoconfirm is disabled
		if !config.Sms.Autoconfirm && config.Hook.SendSMS.Enabled {
			input := hooks.SendSMSInput{
				User: user,
				SMS: hooks.SMS{
					OTP: otp,
				},
			}
			output := hooks.SendSMSOutput{}
			err := a.invokeHook(tx, r, &input, &output, a.config.Hook.SendSMS.URI)
			if err != nil {
				return "", err
			}
		} else {
			messageID, err = smsProvider.SendMessage(phone, message, channel, otp)
			if err != nil {
				return messageID, err
			}
		}
	}

	token := crypto.GenerateTokenHash(phone, otp)

	switch otpType {
	case phoneConfirmationOtp:
		user.ConfirmationSentAt = &now
	case phoneChangeVerification:
		user.PhoneChangeSentAt = &now
	case phoneReauthenticationOtp:
		user.ReauthenticationSentAt = &now
	}

	if err := tx.UpdateOnly(user, includeFields...); err != nil {
		return messageID, errors.Wrap(err, "Database error updating user for phone")
	}

	switch otpType {
	case phoneConfirmationOtp:
		if _, err := models.CreateOneTimeToken(tx, user.ID, user.GetPhone(), token, models.ConfirmationToken); err != nil {
			return messageID, errors.Wrap(err, "Database error creating confirmation token for phone")
		}
	case phoneChangeVerification:
		if _, err := models.CreateOneTimeToken(tx, user.ID, user.PhoneChange, token, models.PhoneChangeToken); err != nil {
			return messageID, errors.Wrap(err, "Database error creating phone change token")
		}
	case phoneReauthenticationOtp:
		if _, err := models.CreateOneTimeToken(tx, user.ID, user.GetPhone(), token, models.ReauthenticationToken); err != nil {
			return messageID, errors.Wrap(err, "Database error creating reauthentication token for phone")
		}
	}

	return messageID, nil
}

func generateSMSFromTemplate(SMSTemplate *template.Template, otp string) (string, error) {
	var message bytes.Buffer
	if err := SMSTemplate.Execute(&message, struct {
		Code string
	}{Code: otp}); err != nil {
		return "", err
	}
	return message.String(), nil
}
