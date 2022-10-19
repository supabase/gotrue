package provider

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const (
	defaultWorkOSAPIBase = "api.workos.com"
)

type workosProvider struct {
	*oauth2.Config
	APIPath         string
	AuthCodeOptions []oauth2.AuthCodeOption
}

// See https://workos.com/docs/reference/sso/profile.
type workosUser struct {
	ID             string                 `mapstructure:"id"`
	ConnectionID   string                 `mapstructure:"connection_id"`
	OrganizationID string                 `mapstructure:"organization_id"`
	ConnectionType string                 `mapstructure:"connection_type"`
	Email          string                 `mapstructure:"email"`
	FirstName      string                 `mapstructure:"first_name"`
	LastName       string                 `mapstructure:"last_name"`
	Object         string                 `mapstructure:"object"`
	IdpID          string                 `mapstructure:"idp_id"`
	RawAttributes  map[string]interface{} `mapstructure:"raw_attributes"`
}

// NewWorkOSProvider creates a WorkOS account provider.
func NewWorkOSProvider(ext conf.OAuthProviderConfiguration, proxyURL string, query *url.Values) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}
	apiPath := chooseHost(ext.URL, defaultWorkOSAPIBase)

	// Attach custom query parameters to the WorkOS authorization URL.
	// See https://workos.com/docs/reference/sso/authorize/get.
	var authCodeOptions []oauth2.AuthCodeOption
	if query != nil {
		if connection := query.Get("connection"); connection != "" {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("connection", connection))
		} else if organization := query.Get("organization"); organization != "" {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("organization", organization))
		} else if provider := query.Get("workos_provider"); provider != "" {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("provider", provider))
		}

		if login_hint := query.Get("login_hint"); login_hint != "" {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("login_hint", login_hint))
		}
	}

	redirectURL := proxyURL

	if redirectURL == "" {
		redirectURL = ext.RedirectURI
	}

	return &workosProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  apiPath + "/sso/authorize",
				TokenURL: apiPath + "/sso/token",
			},
			RedirectURL: redirectURL,
		},
		APIPath:         apiPath,
		AuthCodeOptions: authCodeOptions,
	}, nil
}

func (g workosProvider) AuthCodeURL(state string, args ...oauth2.AuthCodeOption) string {
	opts := append(args, g.AuthCodeOptions...)
	return g.Config.AuthCodeURL(state, opts...)
}

func (g workosProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(context.Background(), code)
}

func (g workosProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	if tok.AccessToken == "" {
		return &UserProvidedData{}, nil
	}

	// WorkOS API returns the user's profile data along with the OAuth2 token, so
	// we can just convert from `map[string]interface{}` to `workosUser` without
	// an additional network request.
	var u workosUser
	err := mapstructure.Decode(tok.Extra("profile"), &u)
	if err != nil {
		return nil, err
	}

	if u.Email == "" {
		return nil, errors.New("unable to find email with WorkOS provider")
	}

	return &UserProvidedData{
		Metadata: &Claims{
			Issuer:        g.APIPath,
			Subject:       u.ID,
			Name:          strings.TrimSpace(u.FirstName + " " + u.LastName),
			Email:         u.Email,
			EmailVerified: true,
			CustomClaims: map[string]interface{}{
				"connection_id":   u.ConnectionID,
				"organization_id": u.OrganizationID,
			},

			// To be deprecated
			FullName:   strings.TrimSpace(u.FirstName + " " + u.LastName),
			ProviderId: u.ID,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: true,
			Primary:  true,
		}},
	}, nil
}
