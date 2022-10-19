package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const defaultSlackApiBase = "slack.com"

type slackProvider struct {
	*oauth2.Config
	APIPath string
}

type slackUser struct {
	ID        string `json:"https://slack.com/user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"picture"`
	TeamID    string `json:"https://slack.com/team_id"`
}

// NewSlackProvider creates a Slack account provider.
func NewSlackProvider(ext conf.OAuthProviderConfiguration, proxyURL string, scopes string) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	apiPath := chooseHost(ext.URL, defaultSlackApiBase) + "/api"
	authPath := chooseHost(ext.URL, defaultSlackApiBase) + "/oauth"

	oauthScopes := []string{
		"profile",
		"email",
		"openid",
	}

	if scopes != "" {
		oauthScopes = append(oauthScopes, strings.Split(scopes, ",")...)
	}

	redirectURL := proxyURL

	if redirectURL == "" {
		redirectURL = ext.RedirectURI
	}

	return &slackProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authPath + "/authorize",
				TokenURL: apiPath + "/oauth.access",
			},
			Scopes:      oauthScopes,
			RedirectURL: redirectURL,
		},
		APIPath: apiPath,
	}, nil
}

func (g slackProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(context.Background(), code)
}

func (g slackProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u slackUser
	if err := makeRequest(ctx, tok, g.Config, g.APIPath+"/openid.connect.userInfo", &u); err != nil {
		return nil, err
	}

	if u.Email == "" {
		return nil, errors.New("unable to find email with Slack provider")
	}

	return &UserProvidedData{
		Metadata: &Claims{
			Issuer:        g.APIPath,
			Subject:       u.ID,
			Name:          u.Name,
			Picture:       u.AvatarURL,
			Email:         u.Email,
			EmailVerified: true, // Slack dosen't provide data on if email is verified.
			CustomClaims: map[string]interface{}{
				"https://slack.com/team_id": u.TeamID,
			},

			// To be deprecated
			AvatarURL:  u.AvatarURL,
			FullName:   u.Name,
			ProviderId: u.ID,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: true, // Slack dosen't provide data on if email is verified.
			Primary:  true,
		}},
	}, nil
}
