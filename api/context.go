package api

import (
	"context"

	jwt "github.com/golang-jwt/jwt"
	"github.com/netlify/gotrue/models"
)

type contextKey string

func (c contextKey) String() string {
	return "gotrue api context key " + string(c)
}

const (
	tokenKey                = contextKey("jwt")
	requestIDKey            = contextKey("request_id")
	inviteTokenKey          = contextKey("invite_token")
	signatureKey            = contextKey("signature")
	externalProviderTypeKey = contextKey("external_provider_type")
	userKey                 = contextKey("user")
	factorKey               = contextKey("factor")
	sessionKey              = contextKey("session")
	externalReferrerKey     = contextKey("external_referrer")
	externalProxyKey        = contextKey("external_proxy")
	functionHooksKey        = contextKey("function_hooks")
	adminUserKey            = contextKey("admin_user")
	oauthTokenKey           = contextKey("oauth_token") // for OAuth1.0, also known as request token
	oauthVerifierKey        = contextKey("oauth_verifier")
)

// withToken adds the JWT token to the context.
func withToken(ctx context.Context, token *jwt.Token) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// getToken reads the JWT token from the context.
func getToken(ctx context.Context) *jwt.Token {
	obj := ctx.Value(tokenKey)
	if obj == nil {
		return nil
	}

	return obj.(*jwt.Token)
}

func getClaims(ctx context.Context) *GoTrueClaims {
	token := getToken(ctx)
	if token == nil {
		return nil
	}
	return token.Claims.(*GoTrueClaims)
}

// withRequestID adds the provided request ID to the context.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// getRequestID reads the request ID from the context.
func getRequestID(ctx context.Context) string {
	obj := ctx.Value(requestIDKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

// withUser adds the user to the context.
func withUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// with Factor adds the factor id to the context.
func withFactor(ctx context.Context, f *models.Factor) context.Context {
	return context.WithValue(ctx, factorKey, f)
}

// getUser reads the user from the context.
func getUser(ctx context.Context) *models.User {
	if ctx == nil {
		return nil
	}
	obj := ctx.Value(userKey)
	if obj == nil {
		return nil
	}
	return obj.(*models.User)
}

// getFactor reads the factor id from the context
func getFactor(ctx context.Context) *models.Factor {
	obj := ctx.Value(factorKey)
	if obj == nil {
		return nil
	}
	return obj.(*models.Factor)
}

// withSession adds the session to the context.
func withSession(ctx context.Context, s *models.Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// getSession reads the session from the context.
func getSession(ctx context.Context) *models.Session {
	if ctx == nil {
		return nil
	}
	obj := ctx.Value(sessionKey)
	if obj == nil {
		return nil
	}
	return obj.(*models.Session)
}

// withSignature adds the provided request ID to the context.
func withSignature(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, signatureKey, id)
}

func withInviteToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, inviteTokenKey, token)
}

func getInviteToken(ctx context.Context) string {
	obj := ctx.Value(inviteTokenKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

// withExternalProviderType adds the provided request ID to the context.
func withExternalProviderType(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, externalProviderTypeKey, id)
}

// getExternalProviderType reads the request ID from the context.
func getExternalProviderType(ctx context.Context) string {
	obj := ctx.Value(externalProviderTypeKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

func withExternalReferrer(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, externalReferrerKey, token)
}

func getExternalReferrer(ctx context.Context) string {
	obj := ctx.Value(externalReferrerKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

func withExternalProxy(ctx context.Context, proxy string) context.Context {
	return context.WithValue(ctx, externalProxyKey, proxy)
}

func getExternalProxy(ctx context.Context) string {
	obj := ctx.Value(externalProxyKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}

// getFunctionHooks reads the request ID from the context.
func getFunctionHooks(ctx context.Context) map[string][]string {
	obj := ctx.Value(functionHooksKey)
	if obj == nil {
		return map[string][]string{}
	}

	return obj.(map[string][]string)
}

// withAdminUser adds the admin user to the context.
func withAdminUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, adminUserKey, u)
}

// getAdminUser reads the admin user from the context.
func getAdminUser(ctx context.Context) *models.User {
	obj := ctx.Value(adminUserKey)
	if obj == nil {
		return nil
	}
	return obj.(*models.User)
}

// withRequestToken adds the request token to the context
func withRequestToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, oauthTokenKey, token)
}

func getRequestToken(ctx context.Context) string {
	obj := ctx.Value(oauthTokenKey)
	if obj == nil {
		return ""
	}
	return obj.(string)
}

func withOAuthVerifier(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, oauthVerifierKey, token)
}

func getOAuthVerifier(ctx context.Context) string {
	obj := ctx.Value(oauthVerifierKey)
	if obj == nil {
		return ""
	}
	return obj.(string)
}
