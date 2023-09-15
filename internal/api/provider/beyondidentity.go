package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/supabase/gotrue/internal/conf"
	"golang.org/x/oauth2"
)

// Beyond Identity
type beyondIdentityProvider struct {
	*oauth2.Config
	Host string
}

type beyondIdentityUser struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

// NewBeyondIdentityProvider creates a BeyondIdentity account provider.
func NewBeyondIdentityProvider(ext conf.OAuthProviderConfiguration, scopes string) (OAuthProvider, error) {

	if err := ext.ValidateOAuth(); err != nil {
		return nil, err
	}

	oauthScopes := []string{
		"email",
		"openid",
	}

	if scopes != "" {
		oauthScopes = append(oauthScopes, strings.Split(scopes, ",")...)
	}

	if ext.URL == "" {
		return nil, errors.New("unable to find Issuer URL for the BeyondIdentity provider")
	}

	extURLlen := len(ext.URL)
	if ext.URL[extURLlen-1] == '/' {
		ext.URL = ext.URL[:extURLlen-1]
	}

	return &beyondIdentityProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID[0],
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  ext.URL + "/authorize",
				TokenURL: ext.URL + "/token",
			},
			RedirectURL: ext.RedirectURI,
			Scopes:      oauthScopes,
		},
		Host: ext.URL,
	}, nil
}

func (g beyondIdentityProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(context.Background(), code)
}

func (g beyondIdentityProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u beyondIdentityUser

	if err := makeRequest(ctx, tok, g.Config, g.Host+"/userinfo", &u); err != nil {
		return nil, err
	}

	var name string
	if u.Name != "" {
		name = u.Name
	} else {
		name = u.Email
	}

	return &UserProvidedData{
		Metadata: &Claims{
			Issuer:            g.Host,
			Subject:           u.Sub,
			Name:              name,
			PreferredUsername: u.PreferredUsername,
			Email:             u.Email,
			EmailVerified:     true, // if email is returned, the email is verified by beyondidentity already

			// To be deprecated
			FullName:   u.Name,
			ProviderId: u.Sub,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: true,
			Primary:  true,
		}},
	}, nil

}
