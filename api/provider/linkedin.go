package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const (
	defaultLinkedinAPIBase = "api.linkedin.com"
)

type linkedinProvider struct {
	*oauth2.Config
	APIPath      string
	UserInfoURL  string
	UserEmailUrl string
}

// See https://docs.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin?context=linkedin/consumer/context
// for retrieving a member's profile. This requires the r_liteprofile scope.
type linkedinUser struct {
	ID        string       `json:"id"`
	FirstName linkedinName `json:"firstName"`
	LastName  linkedinName `json:"lastName"`
	AvatarURL struct {
		DisplayImage struct {
			Elements []struct {
				Identifiers []struct {
					Identifier string `json:"identifier"`
				} `json:"identifiers"`
			} `json:"elements"`
		} `json:"displayImage~"`
	} `json:"profilePicture"`
}

type linkedinName struct {
	Localized       interface{}    `json:"localized"`
	PreferredLocale linkedinLocale `json:"preferredLocale"`
}

type linkedinLocale struct {
	Country  string `json:"country"`
	Language string `json:"language"`
}

// See https://docs.microsoft.com/en-us/linkedin/consumer/integrations/self-serve/sign-in-with-linkedin?context=linkedin/consumer/context#retrieving-member-email-address
// for retrieving a member email address. This requires the r_email_address scope.
type linkedinElements struct {
	Elements []struct {
		Handle      string `json:"handle"`
		HandleTilde struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"handle~"`
	} `json:"elements"`
}

// NewLinkedinProvider creates a Linkedin account provider.
func NewLinkedinProvider(ext conf.OAuthProviderConfiguration, proxyURL string, scopes string) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	apiPath := chooseHost(ext.URL, defaultLinkedinAPIBase)

	oauthScopes := []string{
		"r_emailaddress",
		"r_liteprofile",
	}

	if scopes != "" {
		oauthScopes = append(oauthScopes, strings.Split(scopes, ",")...)
	}

	redirectURL := proxyURL

	if redirectURL == "" {
		redirectURL = ext.RedirectURI
	}

	return &linkedinProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  apiPath + "/oauth/v2/authorization",
				TokenURL: apiPath + "/oauth/v2/accessToken",
			},
			Scopes:      oauthScopes,
			RedirectURL: redirectURL,
		},
		APIPath: apiPath,
	}, nil
}

func (g linkedinProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(context.Background(), code)
}

func GetName(name linkedinName) string {
	key := name.PreferredLocale.Language + "_" + name.PreferredLocale.Country
	myMap := name.Localized.(map[string]interface{})
	return myMap[key].(string)
}

func (g linkedinProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u linkedinUser
	if err := makeRequest(ctx, tok, g.Config, g.APIPath+"/v2/me?projection=(id,firstName,lastName,profilePicture(displayImage~:playableStreams))", &u); err != nil {
		return nil, err
	}

	var e linkedinElements
	// Note: Use primary contact api for handling phone numbers
	if err := makeRequest(ctx, tok, g.Config, g.APIPath+"/v2/emailAddress?q=members&projection=(elements*(handle~))", &e); err != nil {
		return nil, err
	}

	if len(e.Elements) <= 0 {
		return nil, errors.New("unable to find email with Linkedin provider")
	}

	emails := []Email{}

	if e.Elements[0].HandleTilde.EmailAddress != "" {
		// linkedin only returns the primary email which is verified for the r_emailaddress scope.
		emails = append(emails, Email{
			Email:    e.Elements[0].HandleTilde.EmailAddress,
			Primary:  true,
			Verified: true,
		})
	}

	return &UserProvidedData{
		Metadata: &Claims{
			Issuer:        g.APIPath,
			Subject:       u.ID,
			Name:          strings.TrimSpace(GetName(u.FirstName) + " " + GetName(u.LastName)),
			Picture:       u.AvatarURL.DisplayImage.Elements[0].Identifiers[0].Identifier,
			Email:         e.Elements[0].HandleTilde.EmailAddress,
			EmailVerified: true,

			// To be deprecated
			AvatarURL:  u.AvatarURL.DisplayImage.Elements[0].Identifiers[0].Identifier,
			FullName:   strings.TrimSpace(GetName(u.FirstName) + " " + GetName(u.LastName)),
			ProviderId: u.ID,
		},
		Emails: emails,
	}, nil
}
