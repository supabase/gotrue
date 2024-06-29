package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
	"github.com/sethvargo/go-password/password"
	"github.com/supabase/auth/internal/api/provider"
	"github.com/supabase/auth/internal/models"
	"github.com/supabase/auth/internal/observability"
	"github.com/supabase/auth/internal/storage"
)

type AdminUserParams struct {
	Aud          string                 `json:"aud"`
	Role         string                 `json:"role"`
	Email        string                 `json:"email"`
	Phone        string                 `json:"phone"`
	Password     *string                `json:"password"`
	PasswordHash string                 `json:"password_hash"`
	EmailConfirm bool                   `json:"email_confirm"`
	PhoneConfirm bool                   `json:"phone_confirm"`
	UserMetaData map[string]interface{} `json:"user_metadata"`
	AppMetaData  map[string]interface{} `json:"app_metadata"`
	BanDuration  string                 `json:"ban_duration"`
}

type adminUserDeleteParams struct {
	ShouldSoftDelete bool `json:"should_soft_delete"`
}

type adminUserUpdateFactorParams struct {
	FriendlyName string `json:"friendly_name"`
	FactorType   string `json:"factor_type"`
}

type AdminListUsersResponse struct {
	Users []*models.User `json:"users"`
	Aud   string         `json:"aud"`
}

func (a *API) loadUser(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()
	db := a.db.WithContext(ctx)

	userID, err := uuid.FromString(chi.URLParam(r, "user_id"))
	if err != nil {
		return nil, notFoundError(ErrorCodeValidationFailed, "user_id must be an UUID")
	}

	observability.LogEntrySetField(r, "user_id", userID)

	u, err := models.FindUserByID(db, userID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(ErrorCodeUserNotFound, "User not found")
		}
		return nil, internalServerError("Database error loading user").WithInternalError(err)
	}

	return withUser(ctx, u), nil
}

func (a *API) loadFactor(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	factorID, err := uuid.FromString(chi.URLParam(r, "factor_id"))
	if err != nil {
		return nil, notFoundError(ErrorCodeValidationFailed, "factor_id must be an UUID")
	}

	observability.LogEntrySetField(r, "factor_id", factorID)

	f, err := models.FindFactorByFactorID(a.db, factorID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(ErrorCodeMFAFactorNotFound, "Factor not found")
		}
		return nil, internalServerError("Database error loading factor").WithInternalError(err)
	}
	return withFactor(r.Context(), f), nil
}

func (a *API) getAdminParams(r *http.Request) (*AdminUserParams, error) {
	params := &AdminUserParams{}
	if err := retrieveRequestParams(r, params); err != nil {
		return nil, err
	}

	return params, nil
}

// adminUsers responds with a list of all users in a given audience
func (a *API) adminUsers(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	aud := a.requestAud(ctx, r)

	pageParams, err := paginate(r)
	if err != nil {
		return badRequestError(ErrorCodeValidationFailed, "Bad Pagination Parameters: %v", err).WithInternalError(err)
	}

	sortParams, err := sort(r, map[string]bool{models.CreatedAt: true}, []models.SortField{{Name: models.CreatedAt, Dir: models.Descending}})
	if err != nil {
		return badRequestError(ErrorCodeValidationFailed, "Bad Sort Parameters: %v", err)
	}

	filter := r.URL.Query().Get("filter")

	users, err := models.FindUsersInAudience(db, aud, pageParams, sortParams, filter)
	if err != nil {
		return internalServerError("Database error finding users").WithInternalError(err)
	}
	addPaginationHeaders(w, r, pageParams)

	return sendJSON(w, http.StatusOK, AdminListUsersResponse{
		Users: users,
		Aud:   aud,
	})
}

// adminUserGet returns information about a single user
func (a *API) adminUserGet(w http.ResponseWriter, r *http.Request) error {
	user := getUser(r.Context())

	return sendJSON(w, http.StatusOK, user)
}

// adminUserUpdate updates a single user object
func (a *API) adminUserUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config
	user := getUser(ctx)
	adminUser := getAdminUser(ctx)
	params, err := a.getAdminParams(r)
	if err != nil {
		return err
	}

	if params.Email != "" {
		params.Email, err = validateEmail(params.Email)
		if err != nil {
			return err
		}
	}

	if params.Phone != "" {
		params.Phone, err = validatePhone(params.Phone)
		if err != nil {
			return err
		}
	}

	if params.BanDuration != "" {
		duration := time.Duration(0)
		if params.BanDuration != "none" {
			duration, err = time.ParseDuration(params.BanDuration)
			if err != nil {
				return badRequestError(ErrorCodeValidationFailed, "invalid format for ban duration: %v", err)
			}
		}
		if terr := user.Ban(a.db, duration); terr != nil {
			return terr
		}
	}

	if params.Password != nil {
		password := *params.Password

		if err := a.checkPasswordStrength(ctx, password); err != nil {
			return err
		}

		if err := user.SetPassword(ctx, password, config.Security.DBEncryption.Encrypt, config.Security.DBEncryption.EncryptionKeyID, config.Security.DBEncryption.EncryptionKey); err != nil {
			return err
		}
	}

	err = db.Transaction(func(tx *storage.Connection) error {
		if params.Role != "" {
			if terr := user.SetRole(tx, params.Role); terr != nil {
				return terr
			}
		}

		if params.EmailConfirm {
			if terr := user.Confirm(tx); terr != nil {
				return terr
			}
		}

		if params.PhoneConfirm {
			if terr := user.ConfirmPhone(tx); terr != nil {
				return terr
			}
		}

		if params.Password != nil {
			if terr := user.UpdatePassword(tx, nil); terr != nil {
				return terr
			}
		}

		var identities []models.Identity
		if params.Email != "" {
			if identity, terr := models.FindIdentityByIdAndProvider(tx, user.ID.String(), "email"); terr != nil && !models.IsNotFoundError(terr) {
				return terr
			} else if identity == nil {
				// if the user doesn't have an existing email
				// then updating the user's email should create a new email identity
				i, terr := a.createNewIdentity(tx, user, "email", structs.Map(provider.Claims{
					Subject:       user.ID.String(),
					Email:         params.Email,
					EmailVerified: params.EmailConfirm,
				}))
				if terr != nil {
					return terr
				}
				identities = append(identities, *i)
			} else {
				// update the existing email identity
				if terr := identity.UpdateIdentityData(tx, map[string]interface{}{
					"email":          params.Email,
					"email_verified": params.EmailConfirm,
				}); terr != nil {
					return terr
				}
			}
			if user.IsAnonymous && params.EmailConfirm {
				user.IsAnonymous = false
				if terr := tx.UpdateOnly(user, "is_anonymous"); terr != nil {
					return terr
				}
			}

			if terr := user.SetEmail(tx, params.Email); terr != nil {
				return terr
			}
		}

		if params.Phone != "" {
			if identity, terr := models.FindIdentityByIdAndProvider(tx, user.ID.String(), "phone"); terr != nil && !models.IsNotFoundError(terr) {
				return terr
			} else if identity == nil {
				// if the user doesn't have an existing phone
				// then updating the user's phone should create a new phone identity
				identity, terr := a.createNewIdentity(tx, user, "phone", structs.Map(provider.Claims{
					Subject:       user.ID.String(),
					Phone:         params.Phone,
					PhoneVerified: params.PhoneConfirm,
				}))
				if terr != nil {
					return terr
				}
				identities = append(identities, *identity)
			} else {
				// update the existing phone identity
				if terr := identity.UpdateIdentityData(tx, map[string]interface{}{
					"phone":          params.Phone,
					"phone_verified": params.PhoneConfirm,
				}); terr != nil {
					return terr
				}
			}
			if user.IsAnonymous && params.PhoneConfirm {
				user.IsAnonymous = false
				if terr := tx.UpdateOnly(user, "is_anonymous"); terr != nil {
					return terr
				}
			}
			if terr := user.SetPhone(tx, params.Phone); terr != nil {
				return terr
			}
		}
		user.Identities = append(user.Identities, identities...)

		if params.AppMetaData != nil {
			if terr := user.UpdateAppMetaData(tx, params.AppMetaData); terr != nil {
				return terr
			}
		}

		if params.UserMetaData != nil {
			if terr := user.UpdateUserMetaData(tx, params.UserMetaData); terr != nil {
				return terr
			}
		}

		if terr := models.NewAuditLogEntry(r, tx, adminUser, models.UserModifiedAction, "", map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
			"user_phone": user.Phone,
		}); terr != nil {
			return terr
		}
		return nil
	})

	if err != nil {
		return internalServerError("Error updating user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserCreate creates a new user based on the provided data
func (a *API) adminUserCreate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	db := a.db.WithContext(ctx)
	config := a.config

	adminUser := getAdminUser(ctx)
	params, err := a.getAdminParams(r)
	if err != nil {
		return err
	}

	aud := a.requestAud(ctx, r)
	if params.Aud != "" {
		aud = params.Aud
	}

	if params.Email == "" && params.Phone == "" {
		return badRequestError(ErrorCodeValidationFailed, "Cannot create a user without either an email or phone")
	}

	var providers []string
	if params.Email != "" {
		params.Email, err = validateEmail(params.Email)
		if err != nil {
			return err
		}
		if user, err := models.IsDuplicatedEmail(db, params.Email, aud, nil); err != nil {
			return internalServerError("Database error checking email").WithInternalError(err)
		} else if user != nil {
			return unprocessableEntityError(ErrorCodeEmailExists, DuplicateEmailMsg)
		}
		providers = append(providers, "email")
	}

	if params.Phone != "" {
		params.Phone, err = validatePhone(params.Phone)
		if err != nil {
			return err
		}
		if exists, err := models.IsDuplicatedPhone(db, params.Phone, aud); err != nil {
			return internalServerError("Database error checking phone").WithInternalError(err)
		} else if exists {
			return unprocessableEntityError(ErrorCodePhoneExists, "Phone number already registered by another user")
		}
		providers = append(providers, "phone")
	}

	if params.Password != nil && params.PasswordHash != "" {
		return badRequestError(ErrorCodeValidationFailed, "Only a password or a password hash should be provided")
	}

	if (params.Password == nil || *params.Password == "") && params.PasswordHash == "" {
		password, err := password.Generate(64, 10, 0, false, true)
		if err != nil {
			return internalServerError("Error generating password").WithInternalError(err)
		}
		params.Password = &password
	}

	var user *models.User
	if params.PasswordHash != "" {
		user, err = models.NewUserWithPasswordHash(params.Phone, params.Email, params.PasswordHash, aud, params.UserMetaData)
	} else {
		user, err = models.NewUser(params.Phone, params.Email, *params.Password, aud, params.UserMetaData)
	}

	if err != nil {
		return internalServerError("Error creating user").WithInternalError(err)
	}

	user.AppMetaData = map[string]interface{}{
		// TODO: Deprecate "provider" field
		// default to the first provider in the providers slice
		"provider":  providers[0],
		"providers": providers,
	}

	err = db.Transaction(func(tx *storage.Connection) error {
		if terr := tx.Create(user); terr != nil {
			return terr
		}

		var identities []models.Identity
		if user.GetEmail() != "" {
			identity, terr := a.createNewIdentity(tx, user, "email", structs.Map(provider.Claims{
				Subject: user.ID.String(),
				Email:   user.GetEmail(),
			}))

			if terr != nil {
				return terr
			}
			identities = append(identities, *identity)
		}

		if user.GetPhone() != "" {
			identity, terr := a.createNewIdentity(tx, user, "phone", structs.Map(provider.Claims{
				Subject: user.ID.String(),
				Phone:   user.GetPhone(),
			}))

			if terr != nil {
				return terr
			}
			identities = append(identities, *identity)
		}

		user.Identities = identities

		if terr := models.NewAuditLogEntry(r, tx, adminUser, models.UserSignedUpAction, "", map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
			"user_phone": user.Phone,
		}); terr != nil {
			return terr
		}

		role := config.JWT.DefaultGroupName
		if params.Role != "" {
			role = params.Role
		}
		if terr := user.SetRole(tx, role); terr != nil {
			return terr
		}

		if params.AppMetaData != nil {
			if terr := user.UpdateAppMetaData(tx, params.AppMetaData); terr != nil {
				return terr
			}
		}

		if params.EmailConfirm {
			if terr := user.Confirm(tx); terr != nil {
				return terr
			}
		}

		if params.PhoneConfirm {
			if terr := user.ConfirmPhone(tx); terr != nil {
				return terr
			}
		}

		if params.BanDuration != "" {
			duration := time.Duration(0)
			if params.BanDuration != "none" {
				duration, err = time.ParseDuration(params.BanDuration)
				if err != nil {
					return badRequestError(ErrorCodeValidationFailed, "invalid format for ban duration: %v", err)
				}
			}
			if terr := user.Ban(a.db, duration); terr != nil {
				return terr
			}
		}

		return nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "invalid format for ban duration") {
			return err
		}
		return internalServerError("Database error creating new user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserDelete deletes a user
func (a *API) adminUserDelete(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := getUser(ctx)
	adminUser := getAdminUser(ctx)

	var err error
	params := &adminUserDeleteParams{}
	body, err := getBodyBytes(r)
	if err != nil {
		return internalServerError("Could not read body").WithInternalError(err)
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, params); err != nil {
			return badRequestError(ErrorCodeBadJSON, "Could not read params: %v", err)
		}
	} else {
		params.ShouldSoftDelete = false
	}

	err = a.db.Transaction(func(tx *storage.Connection) error {
		if terr := models.NewAuditLogEntry(r, tx, adminUser, models.UserDeletedAction, "", map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
			"user_phone": user.Phone,
		}); terr != nil {
			return internalServerError("Error recording audit log entry").WithInternalError(terr)
		}

		if params.ShouldSoftDelete {
			if user.DeletedAt != nil {
				// user has been soft deleted already
				return nil
			}
			if terr := user.SoftDeleteUser(tx); terr != nil {
				return internalServerError("Error soft deleting user").WithInternalError(terr)
			}

			if terr := user.SoftDeleteUserIdentities(tx); terr != nil {
				return internalServerError("Error soft deleting user identities").WithInternalError(terr)
			}

			// hard delete all associated factors
			if terr := models.DeleteFactorsByUserId(tx, user.ID); terr != nil {
				return internalServerError("Error deleting user's factors").WithInternalError(terr)
			}
			// hard delete all associated sessions
			if terr := models.Logout(tx, user.ID); terr != nil {
				return internalServerError("Error deleting user's sessions").WithInternalError(terr)
			}
		} else {
			if terr := tx.Destroy(user); terr != nil {
				return internalServerError("Database error deleting user").WithInternalError(terr)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, map[string]interface{}{})
}

func (a *API) adminUserDeleteFactor(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := getUser(ctx)
	factor := getFactor(ctx)

	err := a.db.Transaction(func(tx *storage.Connection) error {
		if terr := models.NewAuditLogEntry(r, tx, user, models.DeleteFactorAction, r.RemoteAddr, map[string]interface{}{
			"user_id":   user.ID,
			"factor_id": factor.ID,
		}); terr != nil {
			return terr
		}
		if terr := tx.Destroy(factor); terr != nil {
			return internalServerError("Database error deleting factor").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return sendJSON(w, http.StatusOK, factor)
}

func (a *API) adminUserGetFactors(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := getUser(ctx)
	return sendJSON(w, http.StatusOK, user.Factors)
}

// adminUserUpdate updates a single factor object
func (a *API) adminUserUpdateFactor(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	factor := getFactor(ctx)
	user := getUser(ctx)
	adminUser := getAdminUser(ctx)
	params := &adminUserUpdateFactorParams{}

	if err := retrieveRequestParams(r, params); err != nil {
		return err
	}

	err := a.db.Transaction(func(tx *storage.Connection) error {
		if params.FriendlyName != "" {
			if terr := factor.UpdateFriendlyName(tx, params.FriendlyName); terr != nil {
				return terr
			}
		}
		if params.FactorType != "" {
			if params.FactorType != models.TOTP {
				return badRequestError(ErrorCodeValidationFailed, "Factor Type not valid")
			}
			if terr := factor.UpdateFactorType(tx, params.FactorType); terr != nil {
				return terr
			}
		}

		if terr := models.NewAuditLogEntry(r, tx, adminUser, models.UpdateFactorAction, "", map[string]interface{}{
			"user_id":     user.ID,
			"factor_id":   factor.ID,
			"factor_type": factor.FactorType,
		}); terr != nil {
			return terr
		}
		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, factor)
}
