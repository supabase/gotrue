package api

import (
	"bytes"
	"encoding/json"
	"github.com/aaronarduino/goqrsvg"
	"github.com/ajstarks/svgo"
	"github.com/boombuler/barcode/qr"
	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/metering"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/pquerna/otp/totp"
	"net/http"
	"net/url"
	"time"
)

type EnrollFactorParams struct {
	FriendlyName string `json:"friendly_name"`
	FactorType   string `json:"factor_type"`
	Issuer       string `json:"issuer"`
}

type TOTPObject struct {
	QRCode string `json:"qr_code"`
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

type EnrollFactorResponse struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`
	TOTP TOTPObject
}

type VerifyFactorParams struct {
	ChallengeID uuid.UUID `json:"challenge_id"`
	Code        string    `json:"code"`
}

type ChallengeFactorResponse struct {
	ID        uuid.UUID `json:"id"`
	ExpiresAt int64     `json:"expires_at"`
}

type VerifyFactorResponse struct {
}

type UnenrollFactorResponse struct {
}

type UnenrollFactorParams struct {
	Code string `json:"code"`
}

func (a *API) EnrollFactor(w http.ResponseWriter, r *http.Request) error {
	const DefaultQRSize = 3
	ctx := r.Context()
	user := getUser(ctx)
	config := a.config

	params := &EnrollFactorParams{}
	issuer := ""
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError(err.Error())
	}
	factorType := params.FactorType
	if factorType != models.TOTP {
		return unprocessableEntityError("factorType needs to be TOTP")
	}
	if params.Issuer == "" {
		u, err := url.Parse(config.SiteURL)
		if err != nil {
			return internalServerError("site url is improperly formatted")
		}
		issuer = u.Host
	} else {
		issuer = params.Issuer
	}

	// Read from DB for certainty
	factors, err := models.FindVerifiedFactorsByUser(a.db, user)
	if err != nil {
		return internalServerError("Error validating number of factors in system")
	}
	// Remove this at v2
	if len(factors) >= 1 {
		return forbiddenError("Only one factor can be enrolled at a time, please unenroll to continue")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user.GetEmail(),
	})
	if err != nil {
		return internalServerError("Error generating QR Code secret key").WithInternalError(err)
	}
	var buf bytes.Buffer
	s := svg.New(&buf)
	qrCode, _ := qr.Encode(key.String(), qr.M, qr.Auto)
	qs := goqrsvg.NewQrSVG(qrCode, DefaultQRSize)
	qs.StartQrSVG(s)
	err = qs.WriteQrSVG(s)
	if err != nil {
		return internalServerError("Error writing to QR Code").WithInternalError(err)
	}
	s.End()

	factor, terr := models.NewFactor(user, params.FriendlyName, params.FactorType, models.FactorUnverifiedState, key.Secret())
	if terr != nil {
		return internalServerError("Database error creating factor").WithInternalError(err)
	}
	terr = a.db.Transaction(func(tx *storage.Connection) error {
		if terr = tx.Create(factor); terr != nil {
			return terr
		}
		if terr := models.NewAuditLogEntry(r, tx, user, models.EnrollFactorAction, r.RemoteAddr, nil); terr != nil {
			return terr
		}
		return nil
	})
	if terr != nil {
		return terr
	}

	return sendJSON(w, http.StatusOK, &EnrollFactorResponse{
		ID:   factor.ID,
		Type: models.TOTP,
		TOTP: TOTPObject{
			// See: https://css-tricks.com/probably-dont-base64-svg/
			QRCode: buf.String(),
			Secret: factor.TOTPSecret,
			URI:    key.URL(),
		},
	})
}

func (a *API) ChallengeFactor(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.config

	user := getUser(ctx)
	factor := getFactor(ctx)
	challenge, err := models.NewChallenge(factor)
	if err != nil {
		return internalServerError("Database error creating challenge").WithInternalError(err)
	}

	terr := a.db.Transaction(func(tx *storage.Connection) error {
		if terr := tx.Create(challenge); terr != nil {
			return terr
		}
		if terr := models.NewAuditLogEntry(r, tx, user, models.CreateChallengeAction, r.RemoteAddr, map[string]interface{}{
			"factor_id":     factor.ID,
			"factor_status": factor.Status,
		}); terr != nil {
			return terr
		}
		return nil
	})
	if terr != nil {
		return terr
	}

	creationTime := challenge.CreatedAt
	expiryTime := creationTime.Add(time.Second * time.Duration(config.MFA.ChallengeExpiryDuration))
	return sendJSON(w, http.StatusOK, &ChallengeFactorResponse{
		ID:        challenge.ID,
		ExpiresAt: expiryTime.Unix(),
	})
}

func (a *API) VerifyFactor(w http.ResponseWriter, r *http.Request) error {
	var err error
	ctx := r.Context()
	user := getUser(ctx)
	factor := getFactor(ctx)
	config := a.config

	params := &VerifyFactorParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err = jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Please check the params passed into VerifyFactor: %v", err)
	}

	challenge, err := models.FindChallengeByChallengeID(a.db, params.ChallengeID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding Challenge").WithInternalError(err)
	}

	if challenge.VerifiedAt != nil {
		return badRequestError("Challenge has already been verified")
	}

	valid := totp.Validate(params.Code, factor.TOTPSecret)
	if !valid {
		return unauthorizedError("Invalid TOTP code entered")
	}

	hasExpired := time.Now().After(challenge.CreatedAt.Add(time.Second * time.Duration(config.MFA.ChallengeExpiryDuration)))
	if hasExpired {
		err := a.db.Transaction(func(tx *storage.Connection) error {
			if terr := tx.Destroy(challenge); terr != nil {
				return internalServerError("Database error deleting challenge").WithInternalError(terr)
			}

			return nil
		})
		if err != nil {
			return err
		}
		return badRequestError("%v has expired, please verify against another challenge or create a new challenge.", challenge.ID)
	}
	err = a.db.Transaction(func(tx *storage.Connection) error {
		if err = models.NewAuditLogEntry(r, tx, user, models.VerifyFactorAction, r.RemoteAddr, map[string]interface{}{
			"factor_id":    factor.ID,
			"challenge_id": challenge.ID,
		}); err != nil {
			return err
		}
		if err = challenge.Verify(tx); err != nil {
			return err
		}
		if factor.Status != models.FactorVerifiedState {
			if err = factor.UpdateStatus(tx, models.FactorVerifiedState); err != nil {
				return err
			}
		}
		token, terr := a.issueRefreshToken(ctx, tx, user, models.TOTP, models.GrantParams{
			FactorID: factor.ID,
		})
		if terr != nil {
			return terr
		}
		if terr = a.setCookieTokens(config, token, false, w); terr != nil {
			return internalServerError("Failed to set JWT cookie. %s", terr)
		}
		if terr = models.InvalidateSessionsWithAALLessThan(tx, user.ID, "aal2"); terr != nil {
			return internalServerError("Failed to update sessions. %s", terr)
		}
		return nil
	})
	if err != nil {
		return err
	}
	metering.RecordLogin(string(models.MFACodeLoginAction), user.ID)

	return sendJSON(w, http.StatusOK, &VerifyFactorResponse{})

}

func (a *API) UnenrollFactor(w http.ResponseWriter, r *http.Request) error {
	var err error
	ctx := r.Context()
	user := getUser(ctx)
	factor := getFactor(ctx)
	session := getSession(ctx)

	params := &UnenrollFactorParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err = jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError(err.Error())
	}

	if factor.Status == models.FactorVerifiedState {
		valid := totp.Validate(params.Code, factor.TOTPSecret)
		if !valid {
			return unauthorizedError("Invalid code entered")
		}
	}

	err = a.db.Transaction(func(tx *storage.Connection) error {
		if err = tx.Destroy(factor); err != nil {
			return err
		}
		if err = models.NewAuditLogEntry(r, tx, user, models.UnenrollFactorAction, r.RemoteAddr, map[string]interface{}{
			"user_id":    user.ID,
			"factor_id":  factor.ID,
			"session_id": session.ID,
		}); err != nil {
			return err
		}
		if err = models.InvalidateOtherFactorAssociatedSessions(tx, session.ID, user.ID, factor.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, &UnenrollFactorResponse{})
}