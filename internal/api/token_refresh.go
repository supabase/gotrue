package api

import (
	"context"
	"encoding/json"
	mathRand "math/rand"
	"net/http"
	"time"

	"github.com/supabase/auth/internal/metering"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/utilities"
)

const retryLoopDuration = 5.0

// RefreshTokenGrantParams are the parameters the RefreshTokenGrant method accepts
type RefreshTokenGrantParams struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenGrant implements the refresh_token grant type flow
func (a *API) RefreshTokenGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	db := a.db.WithContext(ctx)
	config := a.config

	params := &RefreshTokenGrantParams{}

	body, err := getBodyBytes(r)
	if err != nil {
		return badRequestError("Could not read body").WithInternalError(err)
	}

	if err := json.Unmarshal(body, params); err != nil {
		return badRequestError("Could not read refresh token grant params: %v", err)
	}

	if params.RefreshToken == "" {
		return oauthError("invalid_request", "refresh_token required")
	}

	// A 5 second retry loop is used to make sure that refresh token
	// requests do not waste database connections waiting for each other.
	// Instead of waiting at the database level, they're waiting at the API
	// level instead and retry to refresh the locked row every 10-30
	// milliseconds.
	retryStart := a.Now()
	retry := true

	for retry && time.Since(retryStart).Seconds() < retryLoopDuration {
		retry = false

		user, token, session, err := models.FindUserWithRefreshToken(db, params.RefreshToken, false)
		if err != nil {
			if models.IsNotFoundError(err) {
				return oauthError("invalid_grant", "Invalid Refresh Token: Refresh Token Not Found")
			}
			return internalServerError(err.Error())
		}

		if user.IsBanned() {
			return oauthError("invalid_grant", "Invalid Refresh Token: User Banned")
		}

		if session != nil {
			result := session.CheckValidity(retryStart, &token.UpdatedAt, config.Sessions.Timebox, config.Sessions.InactivityTimeout)

			switch result {
			case models.SessionValid:
				// do nothing

			case models.SessionTimedOut:
				return oauthError("invalid_grant", "Invalid Refresh Token: Session Expired (Inactivity)")

			default:
				return oauthError("invalid_grant", "Invalid Refresh Token: Session Expired")
			}
		}

		// Basic checks above passed, now we need to serialize access
		// to the session in a transaction so that there's no
		// concurrent modification. In the event that the refresh
		// token's row or session is locked, the transaction is closed
		// and the whole process will be retried a bit later so that
		// the connection pool does not get exhausted.

		var tokenString string
		var expiresAt int64
		var newTokenResponse *AccessTokenResponse

		err = db.Transaction(func(tx *storage.Connection) error {
			user, token, session, terr := models.FindUserWithRefreshToken(tx, params.RefreshToken, true /* forUpdate */)
			if terr != nil {
				if models.IsNotFoundError(terr) {
					// because forUpdate was set, and the
					// previous check outside the
					// transaction found a refresh token
					// and session, but now we're getting a
					// IsNotFoundError, this means that the
					// refresh token row and session are
					// probably locked so we need to retry
					// in a few milliseconds.
					retry = true
					return terr
				}
				return internalServerError(terr.Error())
			}

			if a.config.Sessions.SinglePerUser {
				sessions, terr := models.FindAllSessionsForUser(tx, user.ID, true /* forUpdate */)
				if models.IsNotFoundError(terr) {
					// because forUpdate was set, and the
					// previous check outside the
					// transaction found a user and
					// session, but now we're getting a
					// IsNotFoundError, this means that the
					// user is locked and we need to retry
					// in a few milliseconds
					retry = true
					return terr
				} else if terr != nil {
					return internalServerError(terr.Error())
				}

				sessionTag := session.DetermineTag(config.Sessions.Tags)

				// go through all sessions of the user and
				// check if the current session is the user's
				// most recently refreshed valid session
				for _, s := range sessions {
					if s.ID == session.ID {
						// current session, skip it
						continue
					}

					if s.CheckValidity(retryStart, nil, config.Sessions.Timebox, config.Sessions.InactivityTimeout) != models.SessionValid {
						// session is not valid so it
						// can't be regarded as active
						// on the user
						continue
					}

					if s.DetermineTag(config.Sessions.Tags) != sessionTag {
						// if tags are specified,
						// ignore sessions with a
						// mismatching tag
						continue
					}

					// since token is not the refresh token
					// of s, we can't use it's UpdatedAt
					// time to compare!
					if s.LastRefreshedAt(nil).After(session.LastRefreshedAt(&token.UpdatedAt)) {
						// session is not the most
						// recently active one
						return oauthError("invalid_grant", "Invalid Refresh Token: Session Expired (Revoked by Newer Login)")
					}
				}

				// this session is the user's active session
			}

			// refresh token row and session are locked at this
			// point, cannot be concurrently refreshed

			var issuedToken *models.RefreshToken

			if token.Revoked {
				activeRefreshToken, terr := session.FindCurrentlyActiveRefreshToken(tx)
				if terr != nil && !models.IsNotFoundError(terr) {
					return internalServerError(terr.Error())
				}

				if activeRefreshToken != nil && activeRefreshToken.Parent.String() == token.Token {
					// Token was revoked, but it's the
					// parent of the currently active one.
					// This indicates that the client was
					// not able to store the result when it
					// refreshed token. This case is
					// allowed, provided we return back the
					// active refresh token instead of
					// creating a new one.
					issuedToken = activeRefreshToken
				} else {
					// For a revoked refresh token to be reused, it
					// has to fall within the reuse interval.
					reuseUntil := token.UpdatedAt.Add(
						time.Second * time.Duration(config.Security.RefreshTokenReuseInterval))

					if a.Now().After(reuseUntil) {
						a.clearCookieTokens(config, w)
						// not OK to reuse this token

						if config.Security.RefreshTokenRotationEnabled {
							// Revoke all tokens in token family
							if err := models.RevokeTokenFamily(tx, token); err != nil {
								return internalServerError(err.Error())
							}
						}

						return storage.NewCommitWithError(oauthError("invalid_grant", "Invalid Refresh Token: Already Used").WithInternalMessage("Possible abuse attempt: %v", token.ID))
					}
				}
			}

			if terr = models.NewAuditLogEntry(r, tx, user, models.TokenRefreshedAction, "", nil); terr != nil {
				return terr
			}

			if issuedToken == nil {
				newToken, terr := models.GrantRefreshTokenSwap(r, tx, user, token)
				if terr != nil {
					return terr
				}

				issuedToken = newToken
			}
			sid := ""
			aal := models.AAL1.String()
			amr := []models.AMREntry{}

			if issuedToken.SessionId != nil {
				sessionId := issuedToken.SessionId
				sid = sessionId.String()
				session, terr := models.FindSessionByID(tx, *sessionId, false)
				if terr != nil {
					return terr
				}
				aal, amr, terr = session.CalculateAALAndAMR(user)
				if terr != nil {
					return terr
				}
			}

			tokenString, expiresAt, terr = a.generateAccessToken(ctx, tx, user, sid, models.TokenRefresh, aal, amr)
			if terr != nil {
				httpErr, ok := terr.(*HTTPError)
				if ok {
					return httpErr
				}
				return internalServerError("error generating jwt token").WithInternalError(terr)
			}

			refreshedAt := a.Now()
			session.RefreshedAt = &refreshedAt

			userAgent := r.Header.Get("User-Agent")
			if userAgent != "" {
				session.UserAgent = &userAgent
			} else {
				session.UserAgent = nil
			}

			ipAddress := utilities.GetIPAddress(r)
			if ipAddress != "" {
				session.IP = &ipAddress
			} else {
				session.IP = nil
			}

			if terr := session.UpdateOnlyRefreshInfo(tx); terr != nil {
				return internalServerError("failed to update session information").WithInternalError(terr)
			}

			newTokenResponse = &AccessTokenResponse{
				Token:        tokenString,
				TokenType:    "bearer",
				ExpiresIn:    config.JWT.Exp,
				ExpiresAt:    expiresAt,
				RefreshToken: issuedToken.Token,
				User:         user,
			}
			if terr = a.setCookieTokens(config, newTokenResponse, false, w); terr != nil {
				return internalServerError("Failed to set JWT cookie. %s", terr)
			}

			return nil
		})
		if err == nil {
			// success
			metering.RecordLogin("token", user.ID)
			return sendJSON(w, http.StatusOK, newTokenResponse)
		}

		if err != nil {
			if retry && models.IsNotFoundError(err) {
				// refresh token and session row were likely locked, so
				// we need to wait a moment before retrying the whole
				// process anew
				time.Sleep(time.Duration(10+mathRand.Intn(20)) * time.Millisecond) // #nosec
				continue
			} else {
				return err
			}
		}
	}

	return conflictError("Too many concurrent token refresh requests on the same session or refresh token")
}
