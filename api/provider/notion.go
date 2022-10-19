package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const (
	defaultNotionApiBase = "api.notion.com"
	notionApiVersion     = "2021-08-16"
)

type notionProvider struct {
	*oauth2.Config
	APIPath string
}

type notionUser struct {
	Bot struct {
		Owner struct {
			User struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				AvatarURL string `json:"avatar_url"`
				Person    struct {
					Email string `json:"email"`
				} `json:"person"`
			} `json:"user"`
		} `json:"owner"`
	} `json:"bot"`
}

// NewNotionProvider creates a Notion account provider.
func NewNotionProvider(ext conf.OAuthProviderConfiguration, proxyURL string) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	authHost := chooseHost(ext.URL, defaultNotionApiBase)
	redirectURL := proxyURL

	if redirectURL == "" {
		redirectURL = ext.RedirectURI
	}

	return &notionProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authHost + "/v1/oauth/authorize",
				TokenURL: authHost + "/v1/oauth/token",
			},
			RedirectURL: redirectURL,
		},
		APIPath: authHost,
	}, nil
}

func (g notionProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(context.Background(), code)
}

func (g notionProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u notionUser

	// Perform http request, because we need to set the Notion-Version header
	req, err := http.NewRequest("GET", g.APIPath+"/v1/users/me", nil)

	if err != nil {
		return nil, err
	}

	// set headers
	req.Header.Set("Notion-Version", notionApiVersion)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("a %v error occurred with retrieving user from notion", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &u)
	if err != nil {
		return nil, err
	}

	if u.Bot.Owner.User.Person.Email == "" {
		return nil, errors.New("unable to find email with notion provider")
	}

	return &UserProvidedData{
		Metadata: &Claims{
			Issuer:        g.APIPath,
			Subject:       u.Bot.Owner.User.ID,
			Name:          u.Bot.Owner.User.Name,
			Picture:       u.Bot.Owner.User.AvatarURL,
			Email:         u.Bot.Owner.User.Person.Email,
			EmailVerified: true, // Notion dosen't provide data on if email is verified.

			// To be deprecated
			AvatarURL:  u.Bot.Owner.User.AvatarURL,
			FullName:   u.Bot.Owner.User.Name,
			ProviderId: u.Bot.Owner.User.ID,
		},
		Emails: []Email{{
			Email:    u.Bot.Owner.User.Person.Email,
			Verified: true, // Notion dosen't provide data on if email is verified.
			Primary:  true,
		}},
	}, nil
}
