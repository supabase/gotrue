package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	jwt "github.com/golang-jwt/jwt"
)

const (
	naverResponse             string = `{"resultcode":"resultcode","message":"message","response":{"id":"123","nickname":"Naver Test","name":"Naver Test","email":"naver@example.com","gender":"gender","age":"age","birthday":"birthday","profile_image":"http://example.com/avatar","birthyear":"birthyear","mobile":"mobile"}}`
	naverResponseAnotherEmail string = `{"resultcode":"resultcode","message":"message","response":{"id":"123","nickname":"Naver Test","name":"Naver Test","email":"another@example.com","gender":"gender","age":"age","birthday":"birthday","profile_image":"http://example.com/avatar","birthyear":"birthyear","mobile":"mobile"}}`
	naverResponseNoEmail      string = `{"resultcode":"resultcode","message":"message","response":{"id":"123","nickname":"Naver Test","name":"Naver Test","gender":"gender","age":"age","birthday":"birthday","profile_image":"http://example.com/avatar","birthyear":"birthyear","mobile":"mobile"}}`
)

func (ts *ExternalTestSuite) TestSignupExternalNaver() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=naver", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Naver.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Naver.ClientID, []string{q.Get("client_id")})
	ts.Equal("code", q.Get("response_type"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	ts.Require().NoError(err)

	ts.Equal("naver", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func NaverTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, response string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2.0/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Naver.RedirectURI, r.FormValue("redirect_uri"))
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"naver_token","expires_in":100000}`)
		case "/v1/nid/me":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, response)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown naver oauth call %s", r.URL.Path)
		}
	}))
	ts.Config.External.Naver.URL = server.URL
	return server
}

func (ts *ExternalTestSuite) TestSignupExternalNaver_AuthorizationCode() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponse
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()
	u := performAuthorization(ts, "naver", code, "")
	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "naver@example.com", "Naver Test", "123", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalNaverDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponse
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()

	u := performAuthorization(ts, "naver", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "naver@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalNaverDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponseNoEmail
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()

	u := performAuthorization(ts, "naver", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "naver@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalNaverDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("123", "naver@example.com", "Naver Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponse
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()

	u := performAuthorization(ts, "naver", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "naver@example.com", "Naver Test", "123", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalNaverSuccessWhenMatchingToken() {
	// name and avatar should be populated from Naver API
	ts.createUser("123", "naver@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponse
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()

	u := performAuthorization(ts, "naver", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "naver@example.com", "Naver Test", "123", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalNaverErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponse
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()

	w := performAuthorizationRequest(ts, "naver", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalNaverErrorWhenWrongToken() {
	ts.createUser("123", "naver@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponse
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()

	w := performAuthorizationRequest(ts, "naver", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalNaverErrorWhenEmailDoesntMatch() {
	ts.createUser("123", "naver@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	response := naverResponseAnotherEmail
	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, response)
	defer server.Close()

	u := performAuthorization(ts, "naver", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}

// func (ts *ExternalTestSuite) TestSignupExternalNaverErrorWhenVerifiedFalse() {
// 	tokenCount, userCount := 0, 0
// 	code := "authcode"
// 	emails := `[{"email":"naver@example.com", "primary": true, "verified": false}]`
// 	server := NaverTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
// 	defer server.Close()

// 	u := performAuthorization(ts, "naver", code, "")

// 	v, err := url.ParseQuery(u.Fragment)
// 	ts.Require().NoError(err)
// 	ts.Equal("unauthorized_client", v.Get("error"))
// 	ts.Equal("401", v.Get("error_code"))
// 	ts.Equal("Unverified email with naver", v.Get("error_description"))
// 	assertAuthorizationFailure(ts, u, "", "", "")
// }
