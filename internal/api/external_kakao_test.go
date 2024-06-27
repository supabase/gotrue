package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/supabase/auth/internal/api/provider"
	"github.com/supabase/auth/internal/models"
)

func (ts *ExternalTestSuite) TestSignupExternalKakao() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=kakao", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Kakao.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Kakao.ClientID, []string{q.Get("client_id")})
	ts.Equal("code", q.Get("response_type"))

	claims := ExternalProviderClaims{}
	p := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	ts.Require().NoError(err)

	ts.Equal("kakao", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func KakaoTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, emails string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Kakao.RedirectURI, r.FormValue("redirect_uri"))
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"kakao_token","expires_in":100000}`)
		case "/v2/user/me":
			*userCount++
			var emailList []provider.Email
			if err := json.Unmarshal([]byte(emails), &emailList); err != nil {
				ts.Fail("Invalid email json %s", emails)
			}

			var email *provider.Email

			for i, e := range emailList {
				if len(e.Email) > 0 {
					email = &emailList[i]
					break
				}
			}

			w.Header().Add("Content-Type", "application/json")
			if email != nil {
				fmt.Fprintf(w, `
					{
						"id":123, 
						"kakao_account": {
							"profile": {
								"nickname":"Kakao Test",
								"profile_image_url":"http://example.com/avatar"
							},
							"email": "%v",
							"is_email_valid": %v,
							"is_email_verified": %v,
							"name": "김카카오",
        					"phone_number": "+82 10-1234-5678",
							"birthyear": "1991",
							"birthday": "1231",
        					"gender": "male"
						}
					}`, email.Email, email.Verified, email.Verified)
			} else {
				fmt.Fprint(w, `
					{
						"id":123, 
						"kakao_account": {
							"profile": {
								"nickname":"Kakao Test",
								"profile_image_url":"http://example.com/avatar"
							}
							"name": "김카카오",
        					"phone_number": "+82 10-1234-5678",
							"birthyear": "1991",
							"birthday": "1231",
        					"gender": "male"
						}
					}`)
			}
		default:
			w.WriteHeader(500)
			ts.Fail("unknown kakao oauth call %s", r.URL.Path)
		}
	}))
	ts.Config.External.Kakao.URL = server.URL
	return server
}

func (ts *ExternalTestSuite) TestSignupExternalKakao_AuthorizationCode() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()
	u := performAuthorization(ts, "kakao", code, "")
	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "kakao@example.com", "김카카오", "123", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalKakaoSuccessWithGenderAndBirthdateAndPhone() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "")

	assertAuthorizationSuccessWithGenderAndBirthdateAndPhone(ts, u, tokenCount, userCount, "kakao@example.com", "김카카오", "123", "http://example.com/avatar", "male", "19911231", "+82 10-1234-5678")
}

func (ts *ExternalTestSuite) TestSignupExternalKakaoDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "kakao@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalKakaoDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "kakao@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalKakaoDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("123", "kakao@example.com", "Kakao Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "kakao@example.com", "김카카오", "123", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalKakaoSuccessWhenMatchingToken() {
	// name and avatar should be populated from Kakao API
	ts.createUser("123", "kakao@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "kakao@example.com", "김카카오", "123", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalKakaoErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "kakao", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalKakaoErrorWhenWrongToken() {
	ts.createUser("123", "kakao@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "kakao", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalKakaoErrorWhenEmailDoesntMatch() {
	ts.createUser("123", "kakao@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"other@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}

func (ts *ExternalTestSuite) TestSignupExternalKakaoErrorWhenVerifiedFalse() {
	ts.Config.Mailer.AllowUnverifiedEmailSignIns = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": false}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "")

	assertAuthorizationFailure(ts, u, "Unverified email with kakao. A confirmation email has been sent to your kakao email", "access_denied", "")
}

func (ts *ExternalTestSuite) TestSignupExternalKakaoErrorWhenUserBanned() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"kakao@example.com", "primary": true, "verified": true}]`
	server := KakaoTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "kakao", code, "")
	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "kakao@example.com", "김카카오", "123", "http://example.com/avatar")

	user, err := models.FindUserByEmailAndAudience(ts.API.db, "kakao@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	t := time.Now().Add(24 * time.Hour)
	user.BannedUntil = &t
	require.NoError(ts.T(), ts.API.db.UpdateOnly(user, "banned_until"))

	u = performAuthorization(ts, "kakao", code, "")
	assertAuthorizationFailure(ts, u, "User is banned", "access_denied", "")
}
