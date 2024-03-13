package api

import (
	"errors"
	"net/http"

	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

// RecoverParams holds the parameters for a password recovery request
type RecoverParams struct {
	Email               string `json:"email"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	ResponseType        string `json:"response_type"`
}

func (p *RecoverParams) Validate() error {
	if p.Email == "" {
		return unprocessableEntityError("Password recovery requires an email")
	}
	var err error
	if p.Email, err = validateEmail(p.Email); err != nil {
		return err
	}
	if err := validateCodeFlowParams(p.CodeChallengeMethod, p.CodeChallenge, p.ResponseType); err != nil {
		return err
	}
	return nil
}

// Recover sends a recovery email
func (a *API) Recover(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config
	params := &RecoverParams{}
	if err := retrieveRequestParams(r, params); err != nil {
		return err
	}

	flowType := getFlow(params.CodeChallenge, params.ResponseType)
	if err := params.Validate(); err != nil {
		return err
	}

	var user *models.User
	var err error
	aud := a.requestAud(ctx, r)

	user, err = models.FindUserByEmailAndAudience(db, params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			return sendJSON(w, http.StatusOK, map[string]string{})
		}
		return internalServerError("Unable to process request").WithInternalError(err)
	}
	if isCodeFlow(flowType) {
		if _, err := generateFlowState(a.db, models.Recovery.String(), models.Recovery, params.CodeChallengeMethod, params.CodeChallenge, &(user.ID), flowType); err != nil {
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
		return a.sendPasswordRecovery(tx, user, mailer, config.SMTP.MaxFrequency, referrer, externalURL, config.Mailer.OtpLength, flowType)
	})
	if err != nil {
		if errors.Is(err, MaxFrequencyLimitError) {
			return tooManyRequestsError("For security purposes, you can only request this once every 60 seconds")
		}
		return internalServerError("Unable to process request").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, map[string]string{})
}
