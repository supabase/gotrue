package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

func (ts *ResendTestSuite) TestResendSuccess() {
	// Create user
	u, err := models.NewUser("123456789", "foo@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")

	// Avoid max freq limit error
	now := time.Now().Add(-1 * time.Minute)

	u.ConfirmationToken = "123456"
	u.ConfirmationSentAt = &now
	u.EmailChange = "bar@example.com"
	u.EmailChangeSentAt = &now
	u.EmailChangeTokenCurrent = "123456"
	u.EmailChangeTokenNew = "123456"
	require.NoError(ts.T(), ts.API.db.Create(u), "Error saving new test user")

	cases := []struct {
		desc     string
		params   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			desc: "Resend signup confirmation",
			params: map[string]interface{}{
				"type":  "signup",
				"email": "foo@example.com",
			},
			expected: map[string]interface{}{},
		},
		{
			desc: "Resend email change",
			params: map[string]interface{}{
				"type":  "email_change",
				"email": "foo@example.com",
			},
			expected: map[string]interface{}{},
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
			require.Equal(ts.T(), http.StatusOK, w.Code)

			switch c.params["type"] {
			case signupVerification, emailChangeVerification:
				u, err := models.FindUserByEmailAndAudience(ts.API.db, c.params["email"].(string), ts.Config.JWT.Aud)
				require.NoError(ts.T(), err)
				require.NotEmpty(ts.T(), u)
				if c.params["type"] == signupVerification {
					require.NotEqual(ts.T(), "123456", u.ConfirmationToken)
					require.NotEqual(ts.T(), now, u.ConfirmationSentAt)
				} else if c.params["type"] == emailChangeVerification {
					require.NotEqual(ts.T(), "123456", u.EmailChangeTokenCurrent)
					require.NotEqual(ts.T(), "123456", u.EmailChangeTokenNew)
					require.NotEqual(ts.T(), now, u.EmailChangeSentAt)
				}
			}
		})
	}
}
