package models

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/supabase/auth/internal/storage"

	"github.com/gofrs/uuid"
)

const InvalidCodeChallengeError = "code challenge does not match previously saved code verifier"
const InvalidCodeMethodError = "code challenge method not supported"

type FlowState struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	UserID               *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	FlowType             string     `json:"flow_type" db:"flow_type"`
	AuthCode             string     `json:"auth_code" db:"auth_code"`
	AuthenticationMethod string     `json:"authentication_method" db:"authentication_method"`
	CodeChallenge        string     `json:"code_challenge" db:"code_challenge"`
	CodeChallengeMethod  string     `json:"code_challenge_method" db:"code_challenge_method"`
	ProviderType         string     `json:"provider_type" db:"provider_type"`
	ProviderAccessToken  string     `json:"provider_access_token" db:"provider_access_token"`
	ProviderRefreshToken string     `json:"provider_refresh_token" db:"provider_refresh_token"`
	AuthCodeIssuedAt     *time.Time `json:"auth_code_issued_at" db:"auth_code_issued_at"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

type CodeChallengeMethod int

const (
	SHA256 CodeChallengeMethod = iota
	Plain
)

func (codeChallengeMethod CodeChallengeMethod) String() string {
	switch codeChallengeMethod {
	case SHA256:
		return "s256"
	case Plain:
		return "plain"
	}
	return ""
}

func ParseCodeChallengeMethod(codeChallengeMethod string) (CodeChallengeMethod, error) {
	switch strings.ToLower(codeChallengeMethod) {
	case "s256":
		return SHA256, nil
	case "plain":
		return Plain, nil
	}
	return 0, fmt.Errorf("unsupported code_challenge method %q", codeChallengeMethod)
}

type FlowType int

const (
	PKCEFlow FlowType = iota
	AuthCodeFlow
	ImplicitFlow
)

func (flowType FlowType) String() string {
	switch flowType {
	case PKCEFlow:
		return "pkce"
	case AuthCodeFlow:
		return "code"
	case ImplicitFlow:
		return "implicit"
	}
	return ""
}

func (FlowState) TableName() string {
	tableName := "flow_state"
	return tableName
}

func NewPKCEFlowState(providerType, codeChallenge string, codeChallengeMethod CodeChallengeMethod, authenticationMethod AuthenticationMethod, userID *uuid.UUID) *FlowState {
	id := uuid.Must(uuid.NewV4())
	authCode := uuid.Must(uuid.NewV4())
	flowState := &FlowState{
		ID:                   id,
		ProviderType:         providerType,
		CodeChallenge:        codeChallenge,
		CodeChallengeMethod:  codeChallengeMethod.String(),
		AuthCode:             authCode.String(),
		FlowType:             PKCEFlow.String(),
		AuthenticationMethod: authenticationMethod.String(),
		UserID:               userID,
	}
	return flowState
}

func NewAuthCodeFlowState(providerType string, authenticationMethod AuthenticationMethod, userID *uuid.UUID) *FlowState {
	id := uuid.Must(uuid.NewV4())
	authCode := uuid.Must(uuid.NewV4())
	// TODO: decide whether to swap out placeholder values
	codeChallenge := AuthCodeFlow.String() + "_" + uuid.Must(uuid.NewV4()).String()
	flowState := &FlowState{
		ID:                   id,
		ProviderType:         providerType,
		CodeChallenge:        codeChallenge,
		CodeChallengeMethod:  Plain.String(),
		AuthCode:             authCode.String(),
		FlowType:             AuthCodeFlow.String(),
		AuthenticationMethod: authenticationMethod.String(),
		UserID:               userID,
	}
	return flowState
}

func FindFlowStateByAuthCode(tx *storage.Connection, authCode string) (*FlowState, error) {
	obj := &FlowState{}
	if err := tx.Eager().Q().Where("auth_code = ?", authCode).First(obj); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, FlowStateNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding flow state")
	}

	return obj, nil
}

func FindFlowStateByID(tx *storage.Connection, id string) (*FlowState, error) {
	obj := &FlowState{}
	if err := tx.Eager().Q().Where("id = ?", id).First(obj); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, FlowStateNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding flow state")
	}

	return obj, nil
}

func FindFlowStateByUserID(tx *storage.Connection, id string, authenticationMethod AuthenticationMethod) (*FlowState, error) {
	obj := &FlowState{}
	if err := tx.Eager().Q().Where("user_id = ? and authentication_method = ?", id, authenticationMethod).Last(obj); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, FlowStateNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding flow state")
	}

	return obj, nil
}

func (f *FlowState) VerifyPKCE(codeVerifier string) error {
	switch f.CodeChallengeMethod {
	case SHA256.String():
		hashedCodeVerifier := sha256.Sum256([]byte(codeVerifier))
		encodedCodeVerifier := base64.RawURLEncoding.EncodeToString(hashedCodeVerifier[:])
		if subtle.ConstantTimeCompare([]byte(f.CodeChallenge), []byte(encodedCodeVerifier)) != 1 {
			return errors.New(InvalidCodeChallengeError)
		}
	case Plain.String():
		if subtle.ConstantTimeCompare([]byte(f.CodeChallenge), []byte(codeVerifier)) != 1 {
			return errors.New(InvalidCodeChallengeError)
		}
	default:
		return errors.New(InvalidCodeMethodError)

	}
	return nil
}

func (f *FlowState) IsExpired(expiryDuration time.Duration) bool {
	if f.AuthCodeIssuedAt != nil && f.AuthenticationMethod == MagicLink.String() {
		return time.Now().After(f.AuthCodeIssuedAt.Add(expiryDuration))
	}
	return time.Now().After(f.CreatedAt.Add(expiryDuration))
}

func (f *FlowState) RecordAuthCodeIssuedAtTime(tx *storage.Connection) error {
	issueTime := time.Now()
	f.AuthCodeIssuedAt = &issueTime
	if err := tx.Update(f); err != nil {
		return err
	}
	return nil
}
