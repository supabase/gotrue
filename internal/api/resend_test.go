package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/crypto"
	mail "github.com/supabase/auth/internal/mailer"
	"github.com/supabase/auth/internal/models"
)

type ResendTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.GlobalConfiguration
}

func TestResend(t *testing.T) {
	api, config, err := setupAPIForTest()
	require.NoError(t, err)

	ts := &ResendTestSuite{
		API:    api,
		Config: config,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *ResendTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)
}

func (ts *ResendTestSuite) TestResendValidation() {
	cases := []struct {
		desc     string
		params   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			desc: "Invalid type",
			params: map[string]interface{}{
				"type":  "invalid",
				"email": "foo@example.com",
			},
			expected: map[string]interface{}{
				"code":    http.StatusBadRequest,
				"message": "Missing one of these types: signup, email_change, sms, phone_change",
			},
		},
		{
			desc: "Type & email mismatch",
			params: map[string]interface{}{
				"type":  "sms",
				"email": "foo@example.com",
			},
			expected: map[string]interface{}{
				"code":    http.StatusBadRequest,
				"message": "Type provided requires a phone number",
			},
		},
		{
			desc: "Phone & email change type",
			params: map[string]interface{}{
				"type":  "email_change",
				"phone": "+123456789",
			},
			expected: map[string]interface{}{
				"code":    http.StatusOK,
				"message": nil,
			},
		},
		{
			desc: "Email & phone number provided",
			params: map[string]interface{}{
				"type":  "email_change",
				"phone": "+123456789",
				"email": "foo@example.com",
			},
			expected: map[string]interface{}{
				"code":    http.StatusBadRequest,
				"message": "Only an email address or phone number should be provided.",
			},
		},
	}
	for _, c := range cases {
		ts.Run(c.desc, func() {
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.params))
			req := httptest.NewRequest(http.MethodPost, "http://localhost/resend", &buffer)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), c.expected["code"], w.Code)

			data := make(map[string]interface{})
			require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
			require.Equal(ts.T(), c.expected["message"], data["msg"])
		})
	}

}

func (ts *ResendTestSuite) TestResendSuccess() {
	// Create user
	u, err := models.NewUser("123456789", "foo@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")

	// Avoid max freq limit error
	now := time.Now().Add(-1 * time.Minute)

	// Enable Phone Logoin for phone related tests
	ts.Config.External.Phone.Enabled = true
	// disable secure email change
	ts.Config.Mailer.SecureEmailChangeEnabled = false

	u.ConfirmationSentAt = &now
	u.EmailChange = "bar@example.com"
	u.EmailChangeSentAt = &now
	require.NoError(ts.T(), ts.API.db.Create(u), "Error saving new test user")

	// see sendConfirmation
	token := addFlowPrefixToToken(crypto.GenerateTokenHash(u.GetEmail(), "123456"), models.ImplicitFlow)
	confirmationOtt, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), token, models.ConfirmationToken)
	require.NoError(ts.T(), err)
	emailChangeNewOtt, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.EmailChange, token, models.EmailChangeTokenNew)
	require.NoError(ts.T(), err)

	phoneUser, err := models.NewUser("1234567890", "", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	phoneUser.EmailChange = "bar@example.com"
	phoneUser.EmailChangeSentAt = &now
	require.NoError(ts.T(), ts.API.db.Create(phoneUser), "Error saving new test user")

	// see sendEmailChange
	token = addFlowPrefixToToken(crypto.GenerateTokenHash(phoneUser.EmailChange, "123456"), models.ImplicitFlow)
	phoneUserEmailChangeOtt, err := models.CreateOneTimeToken(ts.API.db, phoneUser.ID, phoneUser.EmailChange, token, models.EmailChangeTokenNew)
	require.NoError(ts.T(), err)

	emailUser, err := models.NewUser("", "bar@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	phoneUser.PhoneChange = "1234567890"
	phoneUser.PhoneChangeSentAt = &now
	require.NoError(ts.T(), ts.API.db.Create(emailUser), "Error saving new test user")

	// see sendPhoneChange
	token = crypto.GenerateTokenHash(phoneUser.PhoneChange, "123456")
	phoneChangeOtt, err := models.CreateOneTimeToken(ts.API.db, phoneUser.ID, phoneUser.PhoneChange, token, models.PhoneChangeToken)
	require.NoError(ts.T(), err)

	cases := []struct {
		desc   string
		params map[string]interface{}
		// expected map[string]interface{}
		user *models.User
		ott  *models.OneTimeToken
	}{
		{
			desc: "Resend signup confirmation",
			params: map[string]interface{}{
				"type":  "signup",
				"email": u.GetEmail(),
			},
			user: u,
			ott:  confirmationOtt,
		},
		{
			desc: "Resend email change",
			params: map[string]interface{}{
				"type":  "email_change",
				"email": u.GetEmail(),
			},
			user: u,
			ott:  emailChangeNewOtt,
		},
		{
			desc: "Resend email change for phone user",
			params: map[string]interface{}{
				"type":  "email_change",
				"phone": phoneUser.GetPhone(),
			},
			user: phoneUser,
			ott:  phoneUserEmailChangeOtt,
		},
		{
			desc: "Resend phone change for email user",
			params: map[string]interface{}{
				"type":  "phone_change",
				"email": emailUser.GetEmail(),
			},
			user: emailUser,
			ott:  phoneChangeOtt,
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.params))

			// resend endpoint is expected to generate a new token to send to the user
			// we need to check that the original token doesn't match the new token
			req := httptest.NewRequest(http.MethodPost, "http://localhost/resend", &buffer)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), http.StatusOK, w.Code)

			switch c.params["type"] {
			case mail.SignupVerification, mail.EmailChangeVerification:
				dbUser, err := models.FindUserByID(ts.API.db, c.user.ID)
				require.NoError(ts.T(), err)
				require.NotEmpty(ts.T(), dbUser)

				if c.params["type"] == mail.SignupVerification {
					require.NotContains(ts.T(), dbUser.OneTimeTokens, c.ott)
					require.NotEqual(ts.T(), dbUser.ConfirmationSentAt, c.user.ConfirmationSentAt)
				} else if c.params["type"] == mail.EmailChangeVerification {
					require.NotContains(ts.T(), dbUser.OneTimeTokens, c.ott)
					require.NotEqual(ts.T(), dbUser.EmailChangeSentAt, c.user.EmailChangeSentAt)
				}
			}
		})
	}
}
