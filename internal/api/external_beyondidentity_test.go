package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	jwt "github.com/golang-jwt/jwt"
)

const (
	beyondIdentityUser string = `{"sub": "beyondidentitytestid", "email": "beyondidentity@example.com", "name": "beyondidentity@example.com", "preferred_username": ""}`
)

func (ts *ExternalTestSuite) TestSignupExternalBeyondIdentity() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=beyondidentity", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.BeyondIdentity.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.BeyondIdentity.ClientID, []string{q.Get("client_id")})
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("email openid", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	ts.Require().NoError(err)

	ts.Equal("beyondidentity", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func BeyondIdentityTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.BeyondIdentity.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"beyondidentity_token","expires_in":100000}`)
		case "/userinfo":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown beyondidentity oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.BeyondIdentity.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalBeyondIdentityWithoutURLSetup() {
	ts.createUser("beyondidentitytestid", "beyondidentity@example.com", "", "", "")
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	ts.Config.External.BeyondIdentity.URL = ""
	defer server.Close()

	w := performAuthorizationRequest(ts, "beyondidentity", code)
	ts.Equal(w.Code, http.StatusBadRequest)
}

func (ts *ExternalTestSuite) TestSignupExternalBeyondIdentity_AuthorizationCode() {
	ts.Config.DisableSignup = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	defer server.Close()

	u := performAuthorization(ts, "beyondidentity", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "beyondidentity@example.com", "beyondidentity@example.com", "beyondidentitytestid", "")
}

func (ts *ExternalTestSuite) TestSignupExternalBeyondIdentityDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	defer server.Close()

	u := performAuthorization(ts, "beyondidentity", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "beyondidentity@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalBeyondIdentityDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("beyondidentitytestid", "beyondidentity@example.com", "beyondidentity@example.com", "avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	defer server.Close()

	u := performAuthorization(ts, "beyondidentity", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "beyondidentity@example.com", "beyondidentity@example.com", "beyondidentitytestid", "avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBeyondIdentitySuccessWhenMatchingToken() {
	ts.createUser("beyondidentitytestid", "beyondidentity@example.com", "beyondidentity@example.com", "avatar", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	defer server.Close()

	u := performAuthorization(ts, "beyondidentity", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "beyondidentity@example.com", "beyondidentity@example.com", "beyondidentitytestid", "avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBeyondIdentityErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	beyondIdentityUser := `{"sub":"beyondidentitytestid","email":"beyondidentity@example.com"}`
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "beyondidentity", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBeyondIdentityErrorWhenWrongToken() {
	ts.createUser("beyondidentitytestid", "beyondidentity@example.com", "beyondidentity@example.com", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	beyondIdentityUser := `{"sub":"beyondidentitytestid","email":"beyondidentity@example.com"}`
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "beyondidentity", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBeyondIdentityErrorWhenEmailDoesntMatch() {
	ts.createUser("beyondidentitytestid", "beyondidentity@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	beyondIdentityUser := `{"sub":"beyondidentitytestid","email":"other@example.com"}`
	server := BeyondIdentityTestSignupSetup(ts, &tokenCount, &userCount, code, beyondIdentityUser)
	defer server.Close()

	u := performAuthorization(ts, "beyondidentity", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}
