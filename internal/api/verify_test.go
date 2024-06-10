package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	mail "github.com/supabase/auth/internal/mailer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/crypto"
	"github.com/supabase/auth/internal/models"
)

type VerifyTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.GlobalConfiguration
}

func TestVerify(t *testing.T) {
	api, config, err := setupAPIForTest()
	require.NoError(t, err)

	ts := &VerifyTestSuite{
		API:    api,
		Config: config,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *VerifyTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)

	// Create user
	u, err := models.NewUser("12345678", "test@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error saving new test user")
}

func (ts *VerifyTestSuite) TestVerifyPasswordRecovery() {
	// modify config so we don't hit rate limit from requesting recovery twice in 60s
	ts.Config.SMTP.MaxFrequency = 60
	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &time.Time{}
	require.NoError(ts.T(), ts.API.db.Update(u))
	testEmail := "test@example.com"

	cases := []struct {
		desc   string
		body   map[string]interface{}
		isPKCE bool
	}{
		{
			desc: "Implict Flow Recovery",
			body: map[string]interface{}{
				"email": testEmail,
			},
			isPKCE: false,
		},
		{
			desc: "PKCE Flow",
			body: map[string]interface{}{
				"email": testEmail,
				// Code Challenge needs to be at least 43 characters long
				"code_challenge":        "6b151854-cc15-4e29-8db7-3d3a9f15b3066b151854-cc15-4e29-8db7-3d3a9f15b306",
				"code_challenge_method": models.SHA256.String(),
			},
			isPKCE: true,
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			// Reset user
			u.EmailConfirmedAt = nil
			require.NoError(ts.T(), ts.API.db.Update(u))
			require.NoError(ts.T(), models.ClearAllOneTimeTokensForUser(ts.API.db, u.ID))

			// Request body
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.body))

			// Setup request
			req := httptest.NewRequest(http.MethodPost, "http://localhost/recover", &buffer)
			req.Header.Set("Content-Type", "application/json")

			// Setup response recorder
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), http.StatusOK, w.Code)

			u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)

			assert.WithinDuration(ts.T(), time.Now(), *u.RecoverySentAt, 1*time.Second)
			assert.False(ts.T(), u.IsConfirmed())

			recoveryToken := u.OneTimeTokens[0].TokenHash

			reqURL := fmt.Sprintf("http://localhost/verify?type=%s&token=%s", mail.RecoveryVerification, recoveryToken)
			req = httptest.NewRequest(http.MethodGet, reqURL, nil)

			w = httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), http.StatusSeeOther, w.Code)

			u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)
			assert.True(ts.T(), u.IsConfirmed())

			if c.isPKCE {
				rURL, _ := w.Result().Location()

				f, err := url.ParseQuery(rURL.RawQuery)
				require.NoError(ts.T(), err)
				assert.NotEmpty(ts.T(), f.Get("code"))
			}
		})
	}
}

func (ts *VerifyTestSuite) TestVerifySecureEmailChange() {
	currentEmail := "test@example.com"
	newEmail := "new@example.com"

	// Change from new email to current email and back to new email
	cases := []struct {
		desc         string
		body         map[string]interface{}
		isPKCE       bool
		currentEmail string
		newEmail     string
	}{
		{
			desc: "Implict Flow Email Change",
			body: map[string]interface{}{
				"email": newEmail,
			},
			isPKCE:       false,
			currentEmail: currentEmail,
			newEmail:     newEmail,
		},
		{
			desc: "PKCE Email Change",
			body: map[string]interface{}{
				"email": currentEmail,
				// Code Challenge needs to be at least 43 characters long
				"code_challenge":        "6b151854-cc15-4e29-8db7-3d3a9f15b3066b151854-cc15-4e29-8db7-3d3a9f15b306",
				"code_challenge_method": models.SHA256.String(),
			},
			isPKCE:       true,
			currentEmail: newEmail,
			newEmail:     currentEmail,
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			u, err := models.FindUserByEmailAndAudience(ts.API.db, c.currentEmail, ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)

			// reset user
			u.EmailChangeSentAt = nil
			require.NoError(ts.T(), ts.API.db.Update(u))
			require.NoError(ts.T(), models.ClearAllOneTimeTokensForUser(ts.API.db, u.ID))

			// Request body
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.body))

			// Setup request
			req := httptest.NewRequest(http.MethodPut, "http://localhost/user", &buffer)
			req.Header.Set("Content-Type", "application/json")

			// Generate access token for request and a mock session
			var token string
			session, err := models.NewSession(u.ID, nil)
			require.NoError(ts.T(), err)
			require.NoError(ts.T(), ts.API.db.Create(session))

			token, _, err = ts.API.generateAccessToken(req, ts.API.db, u, &session.ID, models.MagicLink)
			require.NoError(ts.T(), err)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			// Setup response recorder
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), http.StatusOK, w.Code)

			u, err = models.FindUserByEmailAndAudience(ts.API.db, c.currentEmail, ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)

			var currentTokenHash, newTokenHash string
			for _, ott := range u.OneTimeTokens {
				if ott.TokenType == models.EmailChangeTokenCurrent {
					currentTokenHash = ott.TokenHash
				} else if ott.TokenType == models.EmailChangeTokenNew {
					newTokenHash = ott.TokenHash
				}
			}

			u, err = models.FindUserByEmailAndAudience(ts.API.db, c.currentEmail, ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)

			assert.WithinDuration(ts.T(), time.Now(), *u.EmailChangeSentAt, 1*time.Second)
			assert.False(ts.T(), u.IsConfirmed())

			// Verify new email
			reqURL := fmt.Sprintf("http://localhost/verify?type=%s&token=%s", mail.EmailChangeVerification, newTokenHash)
			req = httptest.NewRequest(http.MethodGet, reqURL, nil)

			w = httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)

			require.Equal(ts.T(), http.StatusSeeOther, w.Code)
			urlVal, err := url.Parse(w.Result().Header.Get("Location"))
			ts.Require().NoError(err, "redirect url parse failed")
			var v url.Values
			if !c.isPKCE {
				v, err = url.ParseQuery(urlVal.Fragment)
				ts.Require().NoError(err)
				ts.Require().NotEmpty(v.Get("message"))
			} else if c.isPKCE {
				v, err = url.ParseQuery(urlVal.RawQuery)
				ts.Require().NoError(err)
				ts.Require().NotEmpty(v.Get("message"))

				v, err = url.ParseQuery(urlVal.Fragment)
				ts.Require().NoError(err)
				ts.Require().NotEmpty(v.Get("message"))
			}

			u, err = models.FindUserByEmailAndAudience(ts.API.db, c.currentEmail, ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)
			assert.Equal(ts.T(), singleConfirmation, u.EmailChangeConfirmStatus)

			// Verify old email
			reqURL = fmt.Sprintf("http://localhost/verify?type=%s&token=%s", mail.EmailChangeVerification, currentTokenHash)
			req = httptest.NewRequest(http.MethodGet, reqURL, nil)

			w = httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), http.StatusSeeOther, w.Code)

			urlVal, err = url.Parse(w.Header().Get("Location"))
			ts.Require().NoError(err, "redirect url parse failed")
			if !c.isPKCE {
				v, err = url.ParseQuery(urlVal.Fragment)
				ts.Require().NoError(err)
				ts.Require().NotEmpty(v.Get("access_token"))
				ts.Require().NotEmpty(v.Get("expires_in"))
				ts.Require().NotEmpty(v.Get("refresh_token"))
			} else if c.isPKCE {
				v, err = url.ParseQuery(urlVal.RawQuery)
				ts.Require().NoError(err)
				ts.Require().NotEmpty(v.Get("code"))
			}

			// user's email should've been updated to newEmail
			u, err = models.FindUserByEmailAndAudience(ts.API.db, c.newEmail, ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)
			require.Equal(ts.T(), zeroConfirmation, u.EmailChangeConfirmStatus)

			// Reset confirmation status after each test
			u.EmailConfirmedAt = nil
			require.NoError(ts.T(), ts.API.db.Update(u))
		})
	}
}

func (ts *VerifyTestSuite) TestExpiredConfirmationToken() {
	// verify variant testing not necessary in this test as it's testing
	// the ConfirmationSentAt behavior, not the ConfirmationToken behavior

	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	token := "asdf3"
	sentTime := time.Now().Add(-48 * time.Hour)
	u.ConfirmationSentAt = &sentTime
	require.NoError(ts.T(), ts.API.db.Update(u))
	ott, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), token, models.ConfirmationToken)
	require.NoError(ts.T(), err)

	// Setup request
	reqURL := fmt.Sprintf("http://localhost/verify?type=%s&token=%s", mail.SignupVerification, ott.TokenHash)
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusSeeOther, w.Code)

	rurl, err := url.Parse(w.Header().Get("Location"))
	require.NoError(ts.T(), err, "redirect url parse failed")

	f, err := url.ParseQuery(rurl.Fragment)
	require.NoError(ts.T(), err)
	assert.Equal(ts.T(), "403", f.Get("error_code"))
	assert.Equal(ts.T(), "Email link is invalid or has expired", f.Get("error_description"))
	assert.Equal(ts.T(), "access_denied", f.Get("error"))
}

func (ts *VerifyTestSuite) TestInvalidOtp() {
	u, err := models.FindUserByPhoneAndAudience(ts.API.db, "12345678", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	sentTime := time.Now().Add(-48 * time.Hour)
	u.ConfirmationSentAt = &sentTime
	u.PhoneChange = "22222222"
	u.PhoneChangeSentAt = &sentTime
	require.NoError(ts.T(), ts.API.db.Update(u))
	confirmationOtt, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "123456", models.ConfirmationToken)
	require.NoError(ts.T(), err)
	phoneChangeOtt, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.PhoneChange, "123456", models.PhoneChangeToken)
	require.NoError(ts.T(), err)

	type ResponseBody struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	expectedResponse := ResponseBody{
		Code: http.StatusForbidden,
		Msg:  "Token has expired or is invalid",
	}

	cases := []struct {
		desc     string
		sentTime time.Time
		body     map[string]interface{}
		expected ResponseBody
	}{
		{
			desc:     "Expired SMS OTP",
			sentTime: time.Now().Add(-48 * time.Hour),
			body: map[string]interface{}{
				"type":  smsVerification,
				"token": confirmationOtt.TokenHash,
				"phone": u.GetPhone(),
			},
			expected: expectedResponse,
		},
		{
			desc:     "Expired Phone Change OTP",
			sentTime: time.Now().Add(-48 * time.Hour),
			body: map[string]interface{}{
				"type":  smsVerification,
				"token": phoneChangeOtt.TokenHash,
				"phone": u.GetPhone(),
			},
			expected: expectedResponse,
		},
		{
			desc:     "Invalid SMS OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  smsVerification,
				"token": "invalid_otp",
				"phone": u.GetPhone(),
			},
			expected: expectedResponse,
		},
		{
			desc:     "Invalid Phone Change OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  phoneChangeVerification,
				"token": "invalid_otp",
				"phone": u.PhoneChange,
			},
			expected: expectedResponse,
		},
		{
			desc:     "Invalid Email OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  mail.SignupVerification,
				"token": "invalid_otp",
				"email": u.GetEmail(),
			},
			expected: expectedResponse,
		},
	}

	for _, caseItem := range cases {
		c := caseItem

		ts.Run(c.desc, func() {
			// update token sent time
			sentTime = time.Now()
			u.ConfirmationSentAt = &c.sentTime
			require.NoError(ts.T(), ts.API.db.Update(u))

			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.body))

			// Setup request
			req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
			req.Header.Set("Content-Type", "application/json")

			// Setup response recorder
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)

			b, err := io.ReadAll(w.Body)
			require.NoError(ts.T(), err)
			var resp ResponseBody
			err = json.Unmarshal(b, &resp)
			require.NoError(ts.T(), err)
			assert.Equal(ts.T(), c.expected.Code, resp.Code)
			assert.Equal(ts.T(), c.expected.Msg, resp.Msg)

		})
	}
}

func (ts *VerifyTestSuite) TestExpiredRecoveryToken() {
	// Since we are testing the RecoverySentAt behavior, not the RecoveryToken behavior, we don't need to create a recovery token
	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	sentTime := time.Now().Add(-48 * time.Hour)
	u.RecoverySentAt = &sentTime
	require.NoError(ts.T(), ts.API.db.Update(u))

	// Setup request
	reqURL := fmt.Sprintf("http://localhost/verify?type=%s&token=%s", "signup", "asdf3")
	req := httptest.NewRequest(http.MethodGet, reqURL, nil)

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusSeeOther, w.Code, w.Body.String())
}

func (ts *VerifyTestSuite) TestVerifyPermitedCustomUri() {
	// verify variant testing not necessary in this test as it's testing
	// the redirect URL behavior, not the RecoveryToken behavior

	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &time.Time{}
	require.NoError(ts.T(), ts.API.db.Update(u))

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/recover", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusOK, w.Code)

	u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	assert.WithinDuration(ts.T(), time.Now(), *u.RecoverySentAt, 1*time.Second)
	assert.False(ts.T(), u.IsConfirmed())

	redirectURL, _ := url.Parse(ts.Config.URIAllowList[0])

	recoveryToken := u.OneTimeTokens[0].TokenHash
	reqURL := fmt.Sprintf("http://localhost/verify?type=%s&token=%s&redirect_to=%s", "recovery", recoveryToken, redirectURL.String())
	req = httptest.NewRequest(http.MethodGet, reqURL, nil)

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusSeeOther, w.Code)
	rURL, _ := w.Result().Location()
	assert.Equal(ts.T(), redirectURL.Hostname(), rURL.Hostname())

	u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.True(ts.T(), u.IsConfirmed())
}

func (ts *VerifyTestSuite) TestVerifyNotPermitedCustomUri() {
	// verify variant testing not necessary in this test as it's testing
	// the redirect URL behavior, not the RecoveryToken behavior

	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &time.Time{}
	require.NoError(ts.T(), ts.API.db.Update(u))

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/recover", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusOK, w.Code)

	u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	assert.WithinDuration(ts.T(), time.Now(), *u.RecoverySentAt, 1*time.Second)
	assert.False(ts.T(), u.IsConfirmed())

	fakeredirectURL, _ := url.Parse("http://custom-url.com")
	siteURL, _ := url.Parse(ts.Config.SiteURL)

	recoveryToken := u.OneTimeTokens[0].TokenHash
	reqURL := fmt.Sprintf("http://localhost/verify?type=%s&token=%s&redirect_to=%s", "recovery", recoveryToken, fakeredirectURL.String())
	req = httptest.NewRequest(http.MethodGet, reqURL, nil)

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusSeeOther, w.Code)
	rURL, _ := w.Result().Location()
	assert.Equal(ts.T(), siteURL.Hostname(), rURL.Hostname())

	u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.True(ts.T(), u.IsConfirmed())
}

func (ts *VerifyTestSuite) TestVerifySignupWithRedirectURLContainedPath() {
	// verify variant testing not necessary in this test as it's testing
	// the redirect URL behavior, not the RecoveryToken behavior

	testCases := []struct {
		desc                string
		siteURL             string
		uriAllowList        []string
		requestredirectURL  string
		expectedredirectURL string
	}{
		{
			desc:                "same site url and redirect url with path",
			siteURL:             "http://localhost:3000/#/",
			uriAllowList:        []string{"http://localhost:3000"},
			requestredirectURL:  "http://localhost:3000/#/",
			expectedredirectURL: "http://localhost:3000/#/",
		},
		{
			desc:                "different site url and redirect url in allow list",
			siteURL:             "https://someapp-something.codemagic.app/#/",
			uriAllowList:        []string{"http://localhost:3000"},
			requestredirectURL:  "http://localhost:3000",
			expectedredirectURL: "http://localhost:3000",
		},
		{
			desc:                "different site url and redirect url not in allow list",
			siteURL:             "https://someapp-something.codemagic.app/#/",
			uriAllowList:        []string{"http://localhost:3000"},
			requestredirectURL:  "http://localhost:3000/docs",
			expectedredirectURL: "https://someapp-something.codemagic.app/#/",
		},
		{
			desc:                "same wildcard site url and redirect url in allow list",
			siteURL:             "http://sub.test.dev:3000/#/",
			uriAllowList:        []string{"http://*.test.dev:3000"},
			requestredirectURL:  "http://sub.test.dev:3000/#/",
			expectedredirectURL: "http://sub.test.dev:3000/#/",
		},
		{
			desc:                "different wildcard site url and redirect url in allow list",
			siteURL:             "http://sub.test.dev/#/",
			uriAllowList:        []string{"http://*.other.dev:3000"},
			requestredirectURL:  "http://sub.other.dev:3000",
			expectedredirectURL: "http://sub.other.dev:3000",
		},
		{
			desc:                "different wildcard site url and redirect url not in allow list",
			siteURL:             "http://test.dev:3000/#/",
			uriAllowList:        []string{"http://*.allowed.dev:3000"},
			requestredirectURL:  "http://sub.test.dev:3000/#/",
			expectedredirectURL: "http://test.dev:3000/#/",
		},
		{
			desc:                "exact mobile deep link redirect url in allow list",
			siteURL:             "http://test.dev:3000/#/",
			uriAllowList:        []string{"twitter://timeline"},
			requestredirectURL:  "twitter://timeline",
			expectedredirectURL: "twitter://timeline",
		},
		// previously the below example was not allowed and with good
		// reason, however users do want flexibility in the redirect
		// URL after the scheme, which is why the example is now corrected
		{
			desc:                "wildcard mobile deep link redirect url in allow list",
			siteURL:             "http://test.dev:3000/#/",
			uriAllowList:        []string{"com.example.app://**"},
			requestredirectURL:  "com.example.app://sign-in/v2",
			expectedredirectURL: "com.example.app://sign-in/v2",
		},
		{
			desc:                "redirect respects . separator",
			siteURL:             "http://localhost:3000",
			uriAllowList:        []string{"http://*.*.dev:3000"},
			requestredirectURL:  "http://foo.bar.dev:3000",
			expectedredirectURL: "http://foo.bar.dev:3000",
		},
		{
			desc:                "redirect does not respect . separator",
			siteURL:             "http://localhost:3000",
			uriAllowList:        []string{"http://*.dev:3000"},
			requestredirectURL:  "http://foo.bar.dev:3000",
			expectedredirectURL: "http://localhost:3000",
		},
		{
			desc:                "redirect respects / separator in url subdirectory",
			siteURL:             "http://localhost:3000",
			uriAllowList:        []string{"http://test.dev:3000/*/*"},
			requestredirectURL:  "http://test.dev:3000/bar/foo",
			expectedredirectURL: "http://test.dev:3000/bar/foo",
		},
		{
			desc:                "redirect does not respect / separator in url subdirectory",
			siteURL:             "http://localhost:3000",
			uriAllowList:        []string{"http://test.dev:3000/*"},
			requestredirectURL:  "http://test.dev:3000/bar/foo",
			expectedredirectURL: "http://localhost:3000",
		},
	}

	for _, tC := range testCases {
		ts.Run(tC.desc, func() {
			// prepare test data
			ts.Config.SiteURL = tC.siteURL
			redirectURL := tC.requestredirectURL
			ts.Config.URIAllowList = tC.uriAllowList
			ts.Config.ApplyDefaults()

			// set verify token to user as it actual do in magic link method
			u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)
			sendTime := time.Now().Add(time.Hour)
			u.ConfirmationSentAt = &sendTime
			require.NoError(ts.T(), ts.API.db.Update(u))
			ott, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "test-confirmation-token", models.ConfirmationToken)
			require.NoError(ts.T(), err)

			reqURL := fmt.Sprintf("http://localhost/verify?type=%s&token=%s&redirect_to=%s", "signup", ott.TokenHash, redirectURL)
			req := httptest.NewRequest(http.MethodGet, reqURL, nil)

			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), http.StatusSeeOther, w.Code)
			rURL, _ := w.Result().Location()
			assert.Contains(ts.T(), rURL.String(), tC.expectedredirectURL) // redirected url starts with per test value

			u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)
			assert.True(ts.T(), u.IsConfirmed())
		})
	}
}

func (ts *VerifyTestSuite) TestVerifyPKCEOTP() {
	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	t := time.Now()
	u.ConfirmationSentAt = &t
	u.RecoverySentAt = &t
	u.EmailChangeSentAt = &t
	require.NoError(ts.T(), ts.API.db.Update(u))

	cases := []struct {
		desc                 string
		payload              *VerifyParams
		authenticationMethod models.AuthenticationMethod
	}{
		{
			desc: "Verify user on signup",
			payload: &VerifyParams{
				Type:  "signup",
				Token: "pkce_confirmation_token",
			},
			authenticationMethod: models.EmailSignup,
		},
		{
			desc: "Verify magiclink",
			payload: &VerifyParams{
				Type:  "magiclink",
				Token: "pkce_recovery_token",
			},
			authenticationMethod: models.MagicLink,
		},
	}
	for _, c := range cases {
		ts.Run(c.desc, func() {
			var buffer bytes.Buffer
			// since the test user is the same, the tokens are being cleared after each successful verification attempt
			// so we create them on each run
			if c.payload.Type == "signup" {
				_, err = models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), c.payload.Token, models.ConfirmationToken)
				require.NoError(ts.T(), err)
			} else if c.payload.Type == "magiclink" {
				_, err = models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), c.payload.Token, models.RecoveryToken)
				require.NoError(ts.T(), err)
			}

			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.payload))
			codeChallenge := "codechallengecodechallengcodechallengcodechallengcodechallenge"
			flowState := models.NewFlowState(c.authenticationMethod.String(), codeChallenge, models.SHA256, c.authenticationMethod, &u.ID)
			require.NoError(ts.T(), ts.API.db.Create(flowState))

			requestUrl := fmt.Sprintf("http://localhost/verify?type=%v&token=%v", c.payload.Type, c.payload.Token)
			req := httptest.NewRequest(http.MethodGet, requestUrl, &buffer)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), http.StatusSeeOther, w.Code)
			rURL, _ := w.Result().Location()

			u, err = models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
			require.NoError(ts.T(), err)
			assert.True(ts.T(), u.IsConfirmed())

			// one time tokens should be cleared after successful verification
			assert.Empty(ts.T(), u.OneTimeTokens)

			f, err := url.ParseQuery(rURL.RawQuery)
			require.NoError(ts.T(), err)
			assert.NotEmpty(ts.T(), f.Get("code"))
		})
	}

}

func (ts *VerifyTestSuite) TestVerifyPostPKCE() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":                 "test@example.com",
		"password":              "test-password",
		"code_challenge":        "codechallengecodechallengcodechallengcodechallengcodechallenge",
		"code_challenge_method": "plain",
	}))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	var data map[string]interface{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	userId, err := uuid.FromString(data["id"].(string))
	require.NoError(ts.T(), err)
	u, err := models.FindUserByID(ts.API.db, userId)
	require.NoError(ts.T(), err)
	require.NotEmpty(ts.T(), u.OneTimeTokens)

	// update the token hash to a known value for subsequent verification
	ott := u.OneTimeTokens[0]
	ott.TokenHash = addFlowPrefixToToken(crypto.GenerateTokenHash("test@example.com", "123456"), models.PKCEFlow)
	require.NoError(ts.T(), ts.API.db.Update(&ott))

	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
		"token": "123456",
		"type":  "email",
	}))
	req = httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	// check that the response contains an access and refresh token
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	require.Contains(ts.T(), data, "access_token")
	require.Contains(ts.T(), data, "refresh_token")

	// check that the user in the db is confirmed
	u, err = models.FindUserByID(ts.API.db, userId)
	require.NoError(ts.T(), err)
	require.True(ts.T(), u.IsConfirmed())

	// // one time tokens should be cleared after successful verification
	require.Empty(ts.T(), u.OneTimeTokens)
}

func (ts *VerifyTestSuite) TestVerifyBannedUser() {
	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	t := time.Now()
	u.ConfirmationSentAt = &t
	u.RecoverySentAt = &t
	u.EmailChangeSentAt = &t

	t = time.Now().Add(24 * time.Hour)
	u.BannedUntil = &t
	require.NoError(ts.T(), ts.API.db.Update(u))
	confirmationOtt, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "test-confirmation-token", models.ConfirmationToken)
	require.NoError(ts.T(), err)
	recoveryOtt, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "test-recovery-token", models.RecoveryToken)
	require.NoError(ts.T(), err)
	emailChangeCurrentOtt, err := models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "test-email-change-token-current", models.EmailChangeTokenCurrent)
	require.NoError(ts.T(), err)

	cases := []struct {
		desc    string
		payload *VerifyParams
	}{
		{
			desc: "Verify banned user on signup",
			payload: &VerifyParams{
				Type:  "signup",
				Token: confirmationOtt.TokenHash,
			},
		},
		{
			desc: "Verify banned user on invite",
			payload: &VerifyParams{
				Type:  "invite",
				Token: confirmationOtt.TokenHash,
			},
		},
		{
			desc: "Verify banned user on recover",
			payload: &VerifyParams{
				Type:  "recovery",
				Token: recoveryOtt.TokenHash,
			},
		},
		{
			desc: "Verify banned user on magiclink",
			payload: &VerifyParams{
				Type:  "magiclink",
				Token: recoveryOtt.TokenHash,
			},
		},
		{
			desc: "Verify banned user on email change",
			payload: &VerifyParams{
				Type:  "email_change",
				Token: emailChangeCurrentOtt.TokenHash,
			},
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.payload))

			requestUrl := fmt.Sprintf("http://localhost/verify?type=%v&token=%v", c.payload.Type, c.payload.Token)
			req := httptest.NewRequest(http.MethodGet, requestUrl, &buffer)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), http.StatusSeeOther, w.Code)

			rurl, err := url.Parse(w.Header().Get("Location"))
			require.NoError(ts.T(), err, "redirect url parse failed")

			f, err := url.ParseQuery(rurl.Fragment)
			require.NoError(ts.T(), err)
			assert.Equal(ts.T(), "403", f.Get("error_code"))
		})
	}
}

func (ts *VerifyTestSuite) TestVerifyValidOtp() {
	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.EmailChange = "new@example.com"
	u.Phone = "12345678"
	u.PhoneChange = "1234567890"
	require.NoError(ts.T(), ts.API.db.Update(u))

	type expected struct {
		code      int
		tokenHash string
	}

	cases := []struct {
		desc     string
		sentTime time.Time
		body     map[string]interface{}
		expected
	}{
		{
			desc:     "Valid SMS OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  smsVerification,
				"token": "123456",
				"phone": u.GetPhone(),
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.GetPhone(), "123456"),
			},
		},
		{
			desc:     "Valid Confirmation OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  mail.SignupVerification,
				"token": "123456",
				"email": u.GetEmail(),
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.GetEmail(), "123456"),
			},
		},
		{
			desc:     "Valid Recovery OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  mail.RecoveryVerification,
				"token": "123456",
				"email": u.GetEmail(),
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.GetEmail(), "123456"),
			},
		},
		{
			desc:     "Valid Email OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  mail.EmailOTPVerification,
				"token": "123456",
				"email": u.GetEmail(),
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.GetEmail(), "123456"),
			},
		},
		{
			desc:     "Valid Email Change OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  mail.EmailChangeVerification,
				"token": "123456",
				"email": u.EmailChange,
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.EmailChange, "123456"),
			},
		},
		{
			desc:     "Valid Phone Change OTP",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":  phoneChangeVerification,
				"token": "123456",
				"phone": u.PhoneChange,
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.PhoneChange, "123456"),
			},
		},
		{
			desc:     "Valid Signup Token Hash",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":       mail.SignupVerification,
				"token_hash": crypto.GenerateTokenHash(u.GetEmail(), "123456"),
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.GetEmail(), "123456"),
			},
		},
		{
			desc:     "Valid Email Change Token Hash",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":       mail.EmailChangeVerification,
				"token_hash": crypto.GenerateTokenHash(u.EmailChange, "123456"),
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.EmailChange, "123456"),
			},
		},
		{
			desc:     "Valid Email Verification Type",
			sentTime: time.Now(),
			body: map[string]interface{}{
				"type":       mail.EmailOTPVerification,
				"token_hash": crypto.GenerateTokenHash(u.GetEmail(), "123456"),
			},
			expected: expected{
				code:      http.StatusOK,
				tokenHash: crypto.GenerateTokenHash(u.GetEmail(), "123456"),
			},
		},
	}

	for _, caseItem := range cases {
		c := caseItem
		ts.Run(c.desc, func() {
			u.ConfirmationSentAt = &c.sentTime
			u.RecoverySentAt = &c.sentTime
			u.EmailChangeSentAt = &c.sentTime
			u.PhoneChangeSentAt = &c.sentTime
			assert.NoError(ts.T(), ts.API.db.Update(u))

			_, err = models.CreateOneTimeToken(ts.API.db, u.ID, "relates_to not used", c.expected.tokenHash, models.ConfirmationToken)
			require.NoError(ts.T(), err)
			_, err = models.CreateOneTimeToken(ts.API.db, u.ID, "relates_to not used", c.expected.tokenHash, models.RecoveryToken)
			require.NoError(ts.T(), err)
			_, err = models.CreateOneTimeToken(ts.API.db, u.ID, "relates_to not used", c.expected.tokenHash, models.EmailChangeTokenNew)
			require.NoError(ts.T(), err)
			_, err = models.CreateOneTimeToken(ts.API.db, u.ID, "relates_to not used", c.expected.tokenHash, models.PhoneChangeToken)
			require.NoError(ts.T(), err)

			require.NoError(ts.T(), ts.API.db.Update(u))

			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.body))

			// Setup request
			req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
			req.Header.Set("Content-Type", "application/json")

			// Setup response recorder
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), c.expected.code, w.Code)
		})
	}
}

func (ts *VerifyTestSuite) TestSecureEmailChangeWithTokenHash() {
	ts.Config.Mailer.SecureEmailChangeEnabled = true
	u, err := models.FindUserByEmailAndAudience(ts.API.db, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.EmailChange = "new@example.com"
	require.NoError(ts.T(), ts.API.db.Update(u))

	currentEmailChangeToken := crypto.GenerateTokenHash(string(u.Email), "123456")
	newEmailChangeToken := crypto.GenerateTokenHash(u.EmailChange, "123456")

	cases := []struct {
		desc                   string
		firstVerificationBody  map[string]interface{}
		secondVerificationBody map[string]interface{}
		expectedStatus         int
	}{
		{
			desc: "Secure Email Change with Token Hash (Success)",
			firstVerificationBody: map[string]interface{}{
				"type":       mail.EmailChangeVerification,
				"token_hash": currentEmailChangeToken,
			},
			secondVerificationBody: map[string]interface{}{
				"type":       mail.EmailChangeVerification,
				"token_hash": newEmailChangeToken,
			},
			expectedStatus: http.StatusOK,
		},
		{
			desc: "Secure Email Change with Token Hash. Reusing a token hash twice should fail",
			firstVerificationBody: map[string]interface{}{
				"type":       mail.EmailChangeVerification,
				"token_hash": currentEmailChangeToken,
			},
			secondVerificationBody: map[string]interface{}{
				"type":       mail.EmailChangeVerification,
				"token_hash": currentEmailChangeToken,
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			// Set the corresponding email change tokens
			require.NoError(ts.T(), models.ClearAllOneTimeTokensForUser(ts.API.db, u.ID))

			_, err = models.CreateOneTimeToken(ts.API.db, u.ID, "relates_to not used", currentEmailChangeToken, models.EmailChangeTokenCurrent)
			require.NoError(ts.T(), err)
			_, err = models.CreateOneTimeToken(ts.API.db, u.ID, "relates_to not used", newEmailChangeToken, models.EmailChangeTokenNew)
			require.NoError(ts.T(), err)

			currentTime := time.Now()
			u.EmailChangeSentAt = &currentTime
			require.NoError(ts.T(), ts.API.db.Update(u))

			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.firstVerificationBody))

			// Setup request
			req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
			req.Header.Set("Content-Type", "application/json")

			// Setup response recorder
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.secondVerificationBody))

			// Setup second request
			req = httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
			req.Header.Set("Content-Type", "application/json")

			// Setup second response recorder
			w = httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), c.expectedStatus, w.Code)
		})
	}
}

func (ts *VerifyTestSuite) TestPrepRedirectURL() {
	escapedMessage := url.QueryEscape(singleConfirmationAccepted)
	cases := []struct {
		desc     string
		message  string
		rurl     string
		flowType models.FlowType
		expected string
	}{
		{
			desc:     "(PKCE): Redirect URL with additional query params",
			message:  singleConfirmationAccepted,
			rurl:     "https://example.com/?first=another&second=other",
			flowType: models.PKCEFlow,
			expected: fmt.Sprintf("https://example.com/?first=another&message=%s&second=other#message=%s", escapedMessage, escapedMessage),
		},
		{
			desc:     "(PKCE): Query params in redirect url are overriden",
			message:  singleConfirmationAccepted,
			rurl:     "https://example.com/?message=Valid+redirect+URL",
			flowType: models.PKCEFlow,
			expected: fmt.Sprintf("https://example.com/?message=%s#message=%s", escapedMessage, escapedMessage),
		},
		{
			desc:     "(Implicit): plain redirect url",
			message:  singleConfirmationAccepted,
			rurl:     "https://example.com/",
			flowType: models.ImplicitFlow,
			expected: fmt.Sprintf("https://example.com/#message=%s", escapedMessage),
		},
		{
			desc:     "(Implicit): query params retained",
			message:  singleConfirmationAccepted,
			rurl:     "https://example.com/?first=another",
			flowType: models.ImplicitFlow,
			expected: fmt.Sprintf("https://example.com/?first=another#message=%s", escapedMessage),
		},
	}
	for _, c := range cases {
		ts.Run(c.desc, func() {
			rurl, err := ts.API.prepRedirectURL(c.message, c.rurl, c.flowType)
			require.NoError(ts.T(), err)
			require.Equal(ts.T(), c.expected, rurl)
		})
	}
}

func (ts *VerifyTestSuite) TestPrepErrorRedirectURL() {
	const DefaultError = "Invalid redirect URL"
	redirectError := fmt.Sprintf("error=invalid_request&error_code=400&error_description=%s", url.QueryEscape(DefaultError))

	cases := []struct {
		desc     string
		message  string
		rurl     string
		flowType models.FlowType
		expected string
	}{
		{
			desc:     "(PKCE): Error in both query params and hash fragment",
			message:  "Valid redirect URL",
			rurl:     "https://example.com/",
			flowType: models.PKCEFlow,
			expected: fmt.Sprintf("https://example.com/?%s#%s", redirectError, redirectError),
		},
		{
			desc:     "(PKCE): Error with conflicting query params in redirect url",
			message:  DefaultError,
			rurl:     "https://example.com/?error=Error+to+be+overriden",
			flowType: models.PKCEFlow,
			expected: fmt.Sprintf("https://example.com/?%s#%s", redirectError, redirectError),
		},
		{
			desc:     "(Implicit): plain redirect url",
			message:  DefaultError,
			rurl:     "https://example.com/",
			flowType: models.ImplicitFlow,
			expected: fmt.Sprintf("https://example.com/#%s", redirectError),
		},
		{
			desc:     "(Implicit): query params preserved",
			message:  DefaultError,
			rurl:     "https://example.com/?test=param",
			flowType: models.ImplicitFlow,
			expected: fmt.Sprintf("https://example.com/?test=param#%s", redirectError),
		},
	}
	for _, c := range cases {
		ts.Run(c.desc, func() {
			req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
			rurl, err := ts.API.prepErrorRedirectURL(badRequestError(ErrorCodeValidationFailed, DefaultError), req, c.rurl, c.flowType)
			require.NoError(ts.T(), err)
			require.Equal(ts.T(), c.expected, rurl)
		})
	}
}

func (ts *VerifyTestSuite) TestVerifyValidateParams() {
	cases := []struct {
		desc     string
		params   *VerifyParams
		method   string
		expected error
	}{
		{
			desc: "Successful GET Verify",
			params: &VerifyParams{
				Type:  "signup",
				Token: "some-token-hash",
			},
			method:   http.MethodGet,
			expected: nil,
		},
		{
			desc: "Successful POST Verify (TokenHash)",
			params: &VerifyParams{
				Type:      "signup",
				TokenHash: "some-token-hash",
			},
			method:   http.MethodPost,
			expected: nil,
		},
		{
			desc: "Successful POST Verify (Token)",
			params: &VerifyParams{
				Type:  "signup",
				Token: "some-token",
				Email: "email@example.com",
			},
			method:   http.MethodPost,
			expected: nil,
		},
		// unsuccessful validations
		{
			desc: "Need to send email or phone number with token",
			params: &VerifyParams{
				Type:  "signup",
				Token: "some-token",
			},
			method:   http.MethodPost,
			expected: badRequestError(ErrorCodeValidationFailed, "Only an email address or phone number should be provided on verify"),
		},
		{
			desc: "Cannot send both TokenHash and Token",
			params: &VerifyParams{
				Type:      "signup",
				Token:     "some-token",
				TokenHash: "some-token-hash",
			},
			method:   http.MethodPost,
			expected: badRequestError(ErrorCodeValidationFailed, "Verify requires either a token or a token hash"),
		},
		{
			desc: "No verification type specified",
			params: &VerifyParams{
				Token: "some-token",
				Email: "email@example.com",
			},
			method:   http.MethodPost,
			expected: badRequestError(ErrorCodeValidationFailed, "Verify requires a verification type"),
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			req := httptest.NewRequest(c.method, "http://localhost", nil)
			err := c.params.Validate(req)
			require.Equal(ts.T(), c.expected, err)
		})
	}
}
