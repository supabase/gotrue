package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/supabase/gotrue/internal/conf"
	"golang.org/x/oauth2"
)

const (
	defaultNaverAuthBase = "nid.naver.com"
	defaultNaverAPIBase  = "openapi.naver.com"
)

type naverProvider struct {
	*oauth2.Config
	APIHost string
}

type naverResponse struct {
	Resultcode string `json:"resultcode"`
	Message    string `json:"message"`
	Response   struct {
		ID           string `json:"id"`
		Nickname     string `json:"nickname"`
		Name         string `json:"name"`
		Email        string `json:"email"`
		Gender       string `json:"gender"`
		Age          string `json:"age"`
		Birthday     string `json:"birthday"`
		ProfileImage string `json:"profile_image"`
		Birthyear    string `json:"birthyear"`
		Mobile       string `json:"mobile"`
	} `json:"response"`
}

func (p naverProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return p.Exchange(context.Background(), code)
}

func (p naverProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var r naverResponse

	if err := makeRequest(ctx, tok, p.Config, p.APIHost+"/v1/nid/me", &r); err != nil {
		return nil, err
	}

	if r.Response.Email == "" {
		return nil, errors.New("unable to find email with Naver provider")
	}

	data := &UserProvidedData{
		Emails: []Email{
			{
				Email:    r.Response.Email,
				Verified: true, // Naver dosen't provide data on if email is verified.
				Primary:  true,
			},
		},
		Metadata: &Claims{
			Issuer:        p.APIHost,
			Subject:       r.Response.ID,
			Email:         r.Response.Email,
			EmailVerified: true, // Naver dosen't provide data on if email is verified.

			Name:              r.Response.Name,
			PreferredUsername: r.Response.Name,

			// To be deprecated
			AvatarURL:   r.Response.ProfileImage,
			FullName:    r.Response.Name,
			ProviderId:  r.Response.ID,
			UserNameKey: r.Response.Name,
		},
	}
	return data, nil
}

func NewNaverProvider(ext conf.OAuthProviderConfiguration, scopes string) (OAuthProvider, error) {
	if err := ext.ValidateOAuth(); err != nil {
		return nil, err
	}

	authHost := chooseHost(ext.URL, defaultNaverAuthBase)
	apiHost := chooseHost(ext.URL, defaultNaverAPIBase)

	oauthScopes := []string{
		"email",
		"profile_image",
	}

	if scopes != "" {
		oauthScopes = append(oauthScopes, strings.Split(scopes, ",")...)
	}

	return &naverProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID[0],
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthStyle: oauth2.AuthStyleInParams,
				AuthURL:   authHost + "/oauth2.0/authorize",
				TokenURL:  authHost + "/oauth2.0/token",
			},
			RedirectURL: ext.RedirectURI,
			Scopes:      oauthScopes,
		},
		APIHost: apiHost,
	}, nil
}
