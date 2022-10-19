package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

// Keycloak
type keycloakProvider struct {
	*oauth2.Config
	Host string
}

type keycloakUser struct {
	Name          string `json:"name"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// NewKeycloakProvider creates a Keycloak account provider.
func NewKeycloakProvider(ext conf.OAuthProviderConfiguration, proxyURL string, scopes string) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	oauthScopes := []string{
		"profile",
		"email",
	}

	if scopes != "" {
		oauthScopes = append(oauthScopes, strings.Split(scopes, ",")...)
	}

	if ext.URL == "" {
		return nil, errors.New("unable to find URL for the Keycloak provider")
	}

	extURLlen := len(ext.URL)
	if ext.URL[extURLlen-1] == '/' {
		ext.URL = ext.URL[:extURLlen-1]
	}

	redirectURL := proxyURL

	if redirectURL == "" {
		redirectURL = ext.RedirectURI
	}

	return &keycloakProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  ext.URL + "/protocol/openid-connect/auth",
				TokenURL: ext.URL + "/protocol/openid-connect/token",
			},
			RedirectURL: redirectURL,
			Scopes:      oauthScopes,
		},
		Host: ext.URL,
	}, nil
}

func (g keycloakProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(context.Background(), code)
}

func (g keycloakProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u keycloakUser

	if err := makeRequest(ctx, tok, g.Config, g.Host+"/protocol/openid-connect/userinfo", &u); err != nil {
		return nil, err
	}

	if u.Email == "" {
		return nil, errors.New("unable to find email with Keycloak provider")
	}

	return &UserProvidedData{
		Metadata: &Claims{
			Issuer:        g.Host,
			Subject:       u.Sub,
			Name:          u.Name,
			Email:         u.Email,
			EmailVerified: u.EmailVerified,

			// To be deprecated
			FullName:   u.Name,
			ProviderId: u.Sub,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: u.EmailVerified,
			Primary:  true,
		}},
	}, nil

}
