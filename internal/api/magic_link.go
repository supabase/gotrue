package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/sethvargo/go-password/password"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

// MagicLinkParams holds the parameters for a magic link request
type MagicLinkParams struct {
	Email               string                 `json:"email"`
	Data                map[string]interface{} `json:"data"`
	CodeChallengeMethod string                 `json:"code_challenge_method"`
	CodeChallenge       string                 `json:"code_challenge"`
	ResponseType        string                 `json:"response_type"`
}

func (p *MagicLinkParams) Validate() error {
	if p.Email == "" {
		return unprocessableEntityError("Password recovery requires an email")
	}
	var err error
	p.Email, err = validateEmail(p.Email)
	if err != nil {
		return err
	}
	if err := validateCodeFlowParams(p.CodeChallengeMethod, p.CodeChallenge, p.ResponseType); err != nil {
		return err
	}
	return nil
}

// MagicLink sends a recovery email
func (a *API) MagicLink(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config

	if !config.External.Email.Enabled {
		return badRequestError("Email logins are disabled")
	}

	params := &MagicLinkParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read verification params: %v", err)
	}

	if err := params.Validate(); err != nil {
		return err
	}

	if params.Data == nil {
		params.Data = make(map[string]interface{})
	}

	flowType := getFlow(params.CodeChallenge, params.ResponseType)

	var isNewUser bool
	aud := a.requestAud(ctx, r)
	user, err := models.FindUserByEmailAndAudience(db, params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			isNewUser = true
		} else {
			return internalServerError("Database error finding user").WithInternalError(err)
		}
	}
	if user != nil {
		isNewUser = !user.IsConfirmed()
	}
	if isNewUser {
		// User either doesn't exist or hasn't completed the signup process.
		// Sign them up with temporary password.
		password, err := password.Generate(64, 10, 1, false, true)
		if err != nil {
			internalServerError("error creating user").WithInternalError(err)
		}

		signUpParams := &SignupParams{
			Email:               params.Email,
			Password:            password,
			Data:                params.Data,
			CodeChallengeMethod: params.CodeChallengeMethod,
			CodeChallenge:       params.CodeChallenge,
			ResponseType:        params.ResponseType,
		}
		newBodyContent, err := json.Marshal(signUpParams)
		if err != nil {
			return badRequestError("Could not parse metadata: %v", err)
		}
		r.Body = io.NopCloser(strings.NewReader(string(newBodyContent)))
		r.ContentLength = int64(len(string(newBodyContent)))

		fakeResponse := &responseStub{}
		if config.Mailer.Autoconfirm {
			// signups are autoconfirmed, send magic link after signup
			if err := a.Signup(fakeResponse, r); err != nil {
				return err
			}
			newBodyContent := &SignupParams{
				Email:               params.Email,
				Data:                params.Data,
				CodeChallengeMethod: params.CodeChallengeMethod,
				CodeChallenge:       params.CodeChallenge,
				ResponseType:        params.ResponseType,
			}
			metadata, err := json.Marshal(newBodyContent)
			if err != nil {
				return badRequestError("Could not parse metadata: %v", err)
			}
			r.Body = io.NopCloser(bytes.NewReader(metadata))
			return a.MagicLink(w, r)
		}
		// otherwise confirmation email already contains 'magic link'
		if err := a.Signup(fakeResponse, r); err != nil {
			return err
		}

		return sendJSON(w, http.StatusOK, make(map[string]string))
	}

	if isCodeFlow(flowType) {
		if _, err = generateFlowState(a.db, models.MagicLink.String(), models.MagicLink, params.CodeChallengeMethod, params.CodeChallenge, &user.ID, flowType); err != nil {
			return err
		}
	}

	err = db.Transaction(func(tx *storage.Connection) error {
		if terr := models.NewAuditLogEntry(r, tx, user, models.UserRecoveryRequestedAction, "", nil); terr != nil {
			return terr
		}
		mailer := a.Mailer(ctx)
		referrer := utilities.GetReferrer(r, config)
		externalURL := getExternalHost(ctx)
		return a.sendMagicLink(tx, user, mailer, config.SMTP.MaxFrequency, referrer, externalURL, config.Mailer.OtpLength, flowType)
	})
	if err != nil {
		if errors.Is(err, MaxFrequencyLimitError) {
			return tooManyRequestsError("For security purposes, you can only request this once every 60 seconds")
		}
		return internalServerError("Error sending magic link").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, make(map[string]string))
}

// responseStub only implement http responsewriter for ignoring
// incoming data from methods where it passed
type responseStub struct {
}

func (rw *responseStub) Header() http.Header {
	return http.Header{}
}

func (rw *responseStub) Write(data []byte) (int, error) {
	return 1, nil
}

func (rw *responseStub) WriteHeader(statusCode int) {
}
