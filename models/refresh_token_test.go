package models

import (
	"net/http"
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RefreshTokenTestSuite struct {
	suite.Suite
	db *storage.Connection
}

func (ts *RefreshTokenTestSuite) SetupTest() {
	TruncateAll(ts.db)
}

func TestRefreshToken(t *testing.T) {
	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	conn, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	ts := &RefreshTokenTestSuite{
		db: conn,
	}
	defer ts.db.Close()

	suite.Run(t, ts)
}

func (ts *RefreshTokenTestSuite) TestGrantAuthenticatedUser() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u, GrantParams{})
	require.NoError(ts.T(), err)

	require.NotEmpty(ts.T(), r.Token)
	require.NotEmpty(ts.T(), r.HashedToken)
	require.NotEqual(ts.T(), r.Token, r.HashedToken)
	require.Equal(ts.T(), r.HashedToken, "H:"+crypto.HashSHA224Base64(r.Token))
	require.Equal(ts.T(), u.ID, r.UserID)
}

func (ts *RefreshTokenTestSuite) TestGrantRefreshTokenSwap() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u, GrantParams{})
	require.NoError(ts.T(), err)
	require.NotEmpty(ts.T(), r.Token)
	require.NotEmpty(ts.T(), r.HashedToken)

	s, err := GrantRefreshTokenSwap(&http.Request{}, ts.db, u, r)
	require.NoError(ts.T(), err)
	require.NotEmpty(ts.T(), s.Token)
	require.NotEmpty(ts.T(), s.HashedToken)
	require.NotEqual(ts.T(), s.Token, s.HashedToken)
	require.Equal(ts.T(), s.Parent, storage.NullString(r.HashedToken))

	_, nr, err := FindUserWithRefreshToken(ts.db, r.Token) // using the original not hashed token
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), nr.Token, r.Token)
	require.NotEmpty(ts.T(), nr.HashedToken)
	require.NotEqual(ts.T(), nr.Token, nr.HashedToken)
	require.Equal(ts.T(), nr.HashedToken, "H:"+crypto.HashSHA224Base64(r.Token))

	require.Equal(ts.T(), r.ID, nr.ID)
	require.True(ts.T(), nr.Revoked, "expected old token to be revoked")

	require.NotEqual(ts.T(), r.ID, s.ID)
	require.Equal(ts.T(), u.ID, s.UserID)
}

func (ts *RefreshTokenTestSuite) TestLogout() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u, GrantParams{})
	require.NoError(ts.T(), err)

	require.NoError(ts.T(), Logout(ts.db, u.ID))
	u, r, err = FindUserWithRefreshToken(ts.db, r.Token) // using the original not hashed token
	require.Errorf(ts.T(), err, "expected error when there are no refresh tokens to authenticate. user: %v token: %v", u, r)

	require.True(ts.T(), IsNotFoundError(err), "expected NotFoundError")
}

func (ts *RefreshTokenTestSuite) createUser() *User {
	return ts.createUserWithEmail("david@netlify.com")
}

func (ts *RefreshTokenTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser("", email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	err = ts.db.Create(user)
	require.NoError(ts.T(), err)

	return user
}
