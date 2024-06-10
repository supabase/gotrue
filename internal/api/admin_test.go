package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/models"
)

type AdminTestSuite struct {
	suite.Suite
	User   *models.User
	API    *API
	Config *conf.GlobalConfiguration

	token string
}

func TestAdmin(t *testing.T) {
	api, config, err := setupAPIForTest()
	require.NoError(t, err)

	ts := &AdminTestSuite{
		API:    api,
		Config: config,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *AdminTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)
	ts.Config.External.Email.Enabled = true
	claims := &AccessTokenClaims{
		Role: "supabase_admin",
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(ts.Config.JWT.Secret))
	require.NoError(ts.T(), err, "Error generating admin jwt")
	ts.token = token
}

// TestAdminUsersUnauthorized tests API /admin/users route without authentication
func (ts *AdminTestSuite) TestAdminUsersUnauthorized() {
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusUnauthorized, w.Code)
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers() {
	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=0>; rel=\"last\"", w.Header().Get("Link"))
	assert.Equal(ts.T(), "0", w.Header().Get("X-Total-Count"))
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_Pagination() {
	u, err := models.NewUser("12345678", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	u, err = models.NewUser("987654321", "test2@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?per_page=1", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=2&per_page=1>; rel=\"next\", </admin/users?page=2&per_page=1>; rel=\"last\"", w.Header().Get("Link"))
	assert.Equal(ts.T(), "2", w.Header().Get("X-Total-Count"))

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	for _, user := range data["users"].([]interface{}) {
		assert.NotEmpty(ts.T(), user)
	}
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_SortAsc() {
	u, err := models.NewUser("", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	u.CreatedAt = time.Now().Add(-time.Minute)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	u, err = models.NewUser("", "test2@example.com", "test", ts.Config.JWT.Aud, nil)
	u.CreatedAt = time.Now()
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	qv := req.URL.Query()
	qv.Set("sort", "created_at asc")
	req.URL.RawQuery = qv.Encode()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 2)
	assert.Equal(ts.T(), "test1@example.com", data.Users[0].GetEmail())
	assert.Equal(ts.T(), "test2@example.com", data.Users[1].GetEmail())
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_SortDesc() {
	u, err := models.NewUser("12345678", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	u.CreatedAt = time.Now().Add(-time.Minute)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	u, err = models.NewUser("987654321", "test2@example.com", "test", ts.Config.JWT.Aud, nil)
	u.CreatedAt = time.Now()
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 2)
	assert.Equal(ts.T(), "test2@example.com", data.Users[0].GetEmail())
	assert.Equal(ts.T(), "test1@example.com", data.Users[1].GetEmail())
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_FilterEmail() {
	u, err := models.NewUser("", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?filter=test1", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 1)
	assert.Equal(ts.T(), "test1@example.com", data.Users[0].GetEmail())
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_FilterName() {
	u, err := models.NewUser("", "test1@example.com", "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": "Test User"})
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	u, err = models.NewUser("", "test2@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?filter=User", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 1)
	assert.Equal(ts.T(), "test1@example.com", data.Users[0].GetEmail())
}

// TestAdminUserCreate tests API /admin/user route (POST)
func (ts *AdminTestSuite) TestAdminUserCreate() {
	cases := []struct {
		desc     string
		params   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			desc: "Only phone",
			params: map[string]interface{}{
				"phone":    "123456789",
				"password": "test1",
			},
			expected: map[string]interface{}{
				"email":           "",
				"phone":           "123456789",
				"isAuthenticated": true,
				"provider":        "phone",
				"providers":       []string{"phone"},
			},
		},
		{
			desc: "With password",
			params: map[string]interface{}{
				"email":    "test1@example.com",
				"phone":    "123456789",
				"password": "test1",
			},
			expected: map[string]interface{}{
				"email":           "test1@example.com",
				"phone":           "123456789",
				"isAuthenticated": true,
				"provider":        "email",
				"providers":       []string{"email", "phone"},
			},
		},
		{
			desc: "Without password",
			params: map[string]interface{}{
				"email": "test2@example.com",
				"phone": "",
			},
			expected: map[string]interface{}{
				"email":           "test2@example.com",
				"phone":           "",
				"isAuthenticated": false,
				"provider":        "email",
				"providers":       []string{"email"},
			},
		},
		{
			desc: "With empty string password",
			params: map[string]interface{}{
				"email":    "test3@example.com",
				"phone":    "",
				"password": "",
			},
			expected: map[string]interface{}{
				"email":           "test3@example.com",
				"phone":           "",
				"isAuthenticated": false,
				"provider":        "email",
				"providers":       []string{"email"},
			},
		},
		{
			desc: "Ban created user",
			params: map[string]interface{}{
				"email":        "test4@example.com",
				"phone":        "",
				"password":     "test1",
				"ban_duration": "24h",
			},
			expected: map[string]interface{}{
				"email":           "test4@example.com",
				"phone":           "",
				"isAuthenticated": true,
				"provider":        "email",
				"providers":       []string{"email"},
			},
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.params))

			// Setup request
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/admin/users", &buffer)

			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))
			ts.Config.External.Phone.Enabled = true

			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), http.StatusOK, w.Code)

			data := models.User{}
			require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
			assert.Equal(ts.T(), c.expected["email"], data.GetEmail())
			assert.Equal(ts.T(), c.expected["phone"], data.GetPhone())
			assert.Equal(ts.T(), c.expected["provider"], data.AppMetaData["provider"])
			assert.ElementsMatch(ts.T(), c.expected["providers"], data.AppMetaData["providers"])

			u, err := models.FindUserByID(ts.API.db, data.ID)
			require.NoError(ts.T(), err)

			// verify that the corresponding identities were created
			require.NotEmpty(ts.T(), u.Identities)
			for _, identity := range u.Identities {
				require.Equal(ts.T(), u.ID, identity.UserID)
				if identity.Provider == "email" {
					require.Equal(ts.T(), c.expected["email"], identity.IdentityData["email"])
				}
				if identity.Provider == "phone" {
					require.Equal(ts.T(), c.expected["phone"], identity.IdentityData["phone"])
				}
			}

			var expectedPassword string
			if _, ok := c.params["password"]; ok {
				expectedPassword = fmt.Sprintf("%v", c.params["password"])
			}

			assert.Equal(ts.T(), c.expected["isAuthenticated"], u.Authenticate(context.Background(), expectedPassword))

			// remove created user after each case
			require.NoError(ts.T(), ts.API.db.Destroy(u))
		})
	}
}

// TestAdminUserGet tests API /admin/user route (GET)
func (ts *AdminTestSuite) TestAdminUserGet() {
	u, err := models.NewUser("12345678", "test1@example.com", "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": "Test Get User"})
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/users/%s", u.ID), nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["email"], "test1@example.com")
	assert.NotNil(ts.T(), data["app_metadata"])
	assert.NotNil(ts.T(), data["user_metadata"])
	md := data["user_metadata"].(map[string]interface{})
	assert.Len(ts.T(), md, 1)
	assert.Equal(ts.T(), "Test Get User", md["full_name"])
}

// TestAdminUserUpdate tests API /admin/user route (UPDATE)
func (ts *AdminTestSuite) TestAdminUserUpdate() {
	u, err := models.NewUser("12345678", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	var buffer bytes.Buffer
	newEmail := "test2@example.com"
	newPhone := "234567890"
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"role": "testing",
		"app_metadata": map[string]interface{}{
			"roles": []string{"writer", "editor"},
		},
		"user_metadata": map[string]interface{}{
			"name": "David",
		},
		"ban_duration": "24h",
		"email":        newEmail,
		"phone":        newPhone,
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/users/%s", u.ID), &buffer)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := models.User{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), "testing", data.Role)
	assert.NotNil(ts.T(), data.UserMetaData)
	assert.Equal(ts.T(), "David", data.UserMetaData["name"])
	assert.Equal(ts.T(), newEmail, data.GetEmail())
	assert.Equal(ts.T(), newPhone, data.GetPhone())

	assert.NotNil(ts.T(), data.AppMetaData)
	assert.Len(ts.T(), data.AppMetaData["roles"], 2)
	assert.Contains(ts.T(), data.AppMetaData["roles"], "writer")
	assert.Contains(ts.T(), data.AppMetaData["roles"], "editor")
	assert.NotNil(ts.T(), data.BannedUntil)

	u, err = models.FindUserByID(ts.API.db, data.ID)
	require.NoError(ts.T(), err)

	// check if the corresponding identities were successfully created
	require.NotEmpty(ts.T(), u.Identities)

	for _, identity := range u.Identities {
		// for email & phone identities, the providerId is the same as the userId
		require.Equal(ts.T(), u.ID.String(), identity.ProviderID)
		require.Equal(ts.T(), u.ID, identity.UserID)
		if identity.Provider == "email" {
			require.Equal(ts.T(), newEmail, identity.IdentityData["email"])
		}
		if identity.Provider == "phone" {
			require.Equal(ts.T(), newPhone, identity.IdentityData["phone"])

		}
	}
}

func (ts *AdminTestSuite) TestAdminUserUpdatePasswordFailed() {
	u, err := models.NewUser("12345678", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	var updateEndpoint = fmt.Sprintf("/admin/users/%s", u.ID)
	ts.Config.Password.MinLength = 6
	ts.Run("Password doesn't meet minimum length", func() {
		var buffer bytes.Buffer
		require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
			"password": "",
		}))

		// Setup request
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, updateEndpoint, &buffer)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

		ts.API.handler.ServeHTTP(w, req)
		require.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code)
	})
}

func (ts *AdminTestSuite) TestAdminUserUpdateBannedUntilFailed() {
	u, err := models.NewUser("", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	var updateEndpoint = fmt.Sprintf("/admin/users/%s", u.ID)
	ts.Config.Password.MinLength = 6
	ts.Run("Incorrect format for ban_duration", func() {
		var buffer bytes.Buffer
		require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
			"ban_duration": "24",
		}))

		// Setup request
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, updateEndpoint, &buffer)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

		ts.API.handler.ServeHTTP(w, req)
		require.Equal(ts.T(), http.StatusBadRequest, w.Code)
	})
}

// TestAdminUserDelete tests API /admin/users route (DELETE)
func (ts *AdminTestSuite) TestAdminUserDelete() {
	type expected struct {
		code int
		err  error
	}
	signupParams := &SignupParams{
		Email:    "test-delete@example.com",
		Password: "test",
		Data:     map[string]interface{}{"name": "test"},
		Provider: "email",
		Aud:      ts.Config.JWT.Aud,
	}
	cases := []struct {
		desc         string
		body         map[string]interface{}
		isSoftDelete string
		isSSOUser    bool
		expected     expected
	}{
		{
			desc:         "Test admin delete user (default)",
			isSoftDelete: "",
			isSSOUser:    false,
			expected:     expected{code: http.StatusOK, err: models.UserNotFoundError{}},
			body:         nil,
		},
		{
			desc:         "Test admin delete user (hard deletion)",
			isSoftDelete: "?is_soft_delete=false",
			isSSOUser:    false,
			expected:     expected{code: http.StatusOK, err: models.UserNotFoundError{}},
			body: map[string]interface{}{
				"should_soft_delete": false,
			},
		},
		{
			desc:         "Test admin delete user (soft deletion)",
			isSoftDelete: "?is_soft_delete=true",
			isSSOUser:    false,
			expected:     expected{code: http.StatusOK, err: models.UserNotFoundError{}},
			body: map[string]interface{}{
				"should_soft_delete": true,
			},
		},
		{
			desc:         "Test admin delete user (soft deletion & sso user)",
			isSoftDelete: "?is_soft_delete=true",
			isSSOUser:    true,
			expected:     expected{code: http.StatusOK, err: nil},
			body: map[string]interface{}{
				"should_soft_delete": true,
			},
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.body))
			u, err := signupParams.ToUserModel(false /* <- isSSOUser */)
			require.NoError(ts.T(), err)
			u, err = ts.API.signupNewUser(ts.API.db, u)
			require.NoError(ts.T(), err)

			// Setup request
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%s", u.ID), &buffer)

			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), c.expected.code, w.Code)

			if c.isSSOUser {
				u, err = models.FindUserByID(ts.API.db, u.ID)
				require.NotNil(ts.T(), u)
			} else {
				_, err = models.FindUserByEmailAndAudience(ts.API.db, signupParams.Email, ts.Config.JWT.Aud)
			}
			require.Equal(ts.T(), c.expected.err, err)
		})
	}
}

func (ts *AdminTestSuite) TestAdminUserSoftDeletion() {
	// create user
	u, err := models.NewUser("123456789", "test@example.com", "secret", ts.Config.JWT.Aud, map[string]interface{}{"name": "test"})
	require.NoError(ts.T(), err)
	u.AppMetaData = map[string]interface{}{
		"provider": "email",
	}
	require.NoError(ts.T(), ts.API.db.Create(u))
	_, err = models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "some_token", models.ConfirmationToken)
	require.NoError(ts.T(), err)
	_, err = models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "some_token", models.RecoveryToken)
	require.NoError(ts.T(), err)
	_, err = models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "some_token", models.EmailChangeTokenCurrent)
	require.NoError(ts.T(), err)
	_, err = models.CreateOneTimeToken(ts.API.db, u.ID, u.GetEmail(), "some_token", models.EmailChangeTokenNew)
	require.NoError(ts.T(), err)
	_, err = models.CreateOneTimeToken(ts.API.db, u.ID, u.GetPhone(), "some_token", models.PhoneChangeToken)
	require.NoError(ts.T(), err)

	// create user identities
	_, err = ts.API.createNewIdentity(ts.API.db, u, "email", map[string]interface{}{
		"sub":   "123456",
		"email": "test@example.com",
	})
	require.NoError(ts.T(), err)
	_, err = ts.API.createNewIdentity(ts.API.db, u, "github", map[string]interface{}{
		"sub":   "234567",
		"email": "test@example.com",
	})
	require.NoError(ts.T(), err)

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"should_soft_delete": true,
	}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%s", u.ID), &buffer)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	// get soft-deleted user from db
	deletedUser, err := models.FindUserByID(ts.API.db, u.ID)
	require.NoError(ts.T(), err)
	require.Empty(ts.T(), deletedUser.EncryptedPassword)

	// one time tokens table should be empty after a soft deletion
	require.Empty(ts.T(), u.OneTimeTokens)
	require.Empty(ts.T(), deletedUser.UserMetaData)
	require.Empty(ts.T(), deletedUser.AppMetaData)
	require.NotEmpty(ts.T(), deletedUser.DeletedAt)
	require.NotEmpty(ts.T(), deletedUser.GetEmail())

	// get soft-deleted user's identity from db
	deletedIdentities, err := models.FindIdentitiesByUserID(ts.API.db, deletedUser.ID)
	require.NoError(ts.T(), err)

	for _, identity := range deletedIdentities {
		require.Empty(ts.T(), identity.IdentityData)
	}
}

func (ts *AdminTestSuite) TestAdminUserCreateWithDisabledLogin() {
	var cases = []struct {
		desc         string
		customConfig *conf.GlobalConfiguration
		userData     map[string]interface{}
		expected     int
	}{
		{
			desc: "Email Signups Disabled",
			customConfig: &conf.GlobalConfiguration{
				JWT: ts.Config.JWT,
				External: conf.ProviderConfiguration{
					Email: conf.EmailProviderConfiguration{
						Enabled: false,
					},
				},
			},
			userData: map[string]interface{}{
				"email":    "test1@example.com",
				"password": "test1",
			},
			expected: http.StatusOK,
		},
		{
			desc: "Phone Signups Disabled",
			customConfig: &conf.GlobalConfiguration{
				JWT: ts.Config.JWT,
				External: conf.ProviderConfiguration{
					Phone: conf.PhoneProviderConfiguration{
						Enabled: false,
					},
				},
			},
			userData: map[string]interface{}{
				"phone":    "123456789",
				"password": "test1",
			},
			expected: http.StatusOK,
		},
		{
			desc: "All Signups Disabled",
			customConfig: &conf.GlobalConfiguration{
				JWT:           ts.Config.JWT,
				DisableSignup: true,
			},
			userData: map[string]interface{}{
				"email":    "test2@example.com",
				"password": "test2",
			},
			expected: http.StatusOK,
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			// Initialize user data
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.userData))

			// Setup request
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/admin/users", &buffer)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

			ts.Config.JWT = c.customConfig.JWT
			ts.Config.External = c.customConfig.External
			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), c.expected, w.Code)
		})
	}
}

// TestAdminUserDeleteFactor tests API /admin/users/<user_id>/factors/<factor_id>/
func (ts *AdminTestSuite) TestAdminUserDeleteFactor() {
	u, err := models.NewUser("123456789", "test-delete@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	f := models.NewFactor(u, "testSimpleName", models.TOTP, models.FactorStateVerified, "secretkey")
	require.NoError(ts.T(), ts.API.db.Create(f), "Error saving new test factor")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%s/factors/%s/", u.ID, f.ID), nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	_, err = models.FindFactorByFactorID(ts.API.db, f.ID)
	require.EqualError(ts.T(), err, models.FactorNotFoundError{}.Error())

}

// TestAdminUserGetFactor tests API /admin/user/<user_id>/factors/
func (ts *AdminTestSuite) TestAdminUserGetFactors() {
	u, err := models.NewUser("123456789", "test-delete@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	f := models.NewFactor(u, "testSimpleName", models.TOTP, models.FactorStateUnverified, "secretkey")
	require.NoError(ts.T(), ts.API.db.Create(f), "Error saving new test factor")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/users/%s/factors/", u.ID), nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)
	getFactorsResp := []*models.Factor{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&getFactorsResp))
	require.Equal(ts.T(), getFactorsResp[0].Secret, "")
}

func (ts *AdminTestSuite) TestAdminUserUpdateFactor() {
	u, err := models.NewUser("123456789", "test-delete@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	f := models.NewFactor(u, "testSimpleName", models.TOTP, models.FactorStateUnverified, "secretkey")
	require.NoError(ts.T(), ts.API.db.Create(f), "Error saving new test factor")

	var cases = []struct {
		Desc         string
		FactorData   map[string]interface{}
		ExpectedCode int
	}{
		{
			Desc: "Update Factor friendly name",
			FactorData: map[string]interface{}{
				"friendly_name": "john",
			},
			ExpectedCode: http.StatusOK,
		},
		{
			Desc: "Update factor: valid factor type",
			FactorData: map[string]interface{}{
				"friendly_name": "john",
				"factor_type":   models.TOTP,
			},
			ExpectedCode: http.StatusOK,
		},
		{
			Desc: "Update factor: invalid factor",
			FactorData: map[string]interface{}{
				"factor_type": "invalid_factor",
			},
			ExpectedCode: http.StatusBadRequest,
		},
	}

	// Initialize factor data
	for _, c := range cases {
		ts.Run(c.Desc, func() {
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(c.FactorData))
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/users/%s/factors/%s/", u.ID, f.ID), &buffer)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))
			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), c.ExpectedCode, w.Code)
		})
	}

}
