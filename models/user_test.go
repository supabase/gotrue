package models

import (
	"strings"
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

const modelsTestConfig = "../hack/test.env"

func init() {
	PasswordHashCost = bcrypt.MinCost
}

type UserTestSuite struct {
	suite.Suite
	db *storage.Connection
}

func (ts *UserTestSuite) SetupTest() {
	TruncateAll(ts.db)
}

func TestUser(t *testing.T) {
	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	conn, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	ts := &UserTestSuite{
		db: conn,
	}
	defer ts.db.Close()

	suite.Run(t, ts)
}

func (ts *UserTestSuite) TestUpdateAppMetadata() {
	u, err := NewUser("", "", "", "", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), u.UpdateAppMetaData(ts.db, make(map[string]interface{})))

	require.NotNil(ts.T(), u.AppMetaData)

	require.NoError(ts.T(), u.UpdateAppMetaData(ts.db, map[string]interface{}{
		"foo": "bar",
	}))

	require.Equal(ts.T(), "bar", u.AppMetaData["foo"])
	require.NoError(ts.T(), u.UpdateAppMetaData(ts.db, map[string]interface{}{
		"foo": nil,
	}))
	require.Len(ts.T(), u.AppMetaData, 0)
	require.Equal(ts.T(), nil, u.AppMetaData["foo"])
}

func (ts *UserTestSuite) TestUpdateUserMetadata() {
	u, err := NewUser("", "", "", "", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), u.UpdateUserMetaData(ts.db, make(map[string]interface{})))

	require.NotNil(ts.T(), u.UserMetaData)

	require.NoError(ts.T(), u.UpdateUserMetaData(ts.db, map[string]interface{}{
		"foo": "bar",
	}))

	require.Equal(ts.T(), "bar", u.UserMetaData["foo"])
	require.NoError(ts.T(), u.UpdateUserMetaData(ts.db, map[string]interface{}{
		"foo": nil,
	}))
	require.Len(ts.T(), u.UserMetaData, 0)
	require.Equal(ts.T(), nil, u.UserMetaData["foo"])
}

func (ts *UserTestSuite) TestFindUserByConfirmationToken() {
	u := ts.createUser()

	n, err := FindUserByConfirmationToken(ts.db, u.ConfirmationToken)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByEmailAndAudience() {
	u := ts.createUser()

	n, err := FindUserByEmailAndAudience(ts.db, u.GetEmail(), "test")
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)

	_, err = FindUserByEmailAndAudience(ts.db, u.GetEmail(), "invalid")
	require.EqualError(ts.T(), err, UserNotFoundError{}.Error())
}

func (ts *UserTestSuite) TestFindUsersInAudience() {
	u := ts.createUser()

	n, err := FindUsersInAudience(ts.db, u.Aud, nil, nil, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)

	p := Pagination{
		Page:    1,
		PerPage: 50,
	}
	n, err = FindUsersInAudience(ts.db, u.Aud, &p, nil, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)
	assert.Equal(ts.T(), uint64(1), p.Count)

	sp := &SortParams{
		Fields: []SortField{
			{Name: "created_at", Dir: Descending},
		},
	}
	n, err = FindUsersInAudience(ts.db, u.Aud, nil, sp, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)
}

func (ts *UserTestSuite) TestFindUserByID() {
	u := ts.createUser()

	n, err := FindUserByID(ts.db, u.ID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByRecoveryToken() {
	u := ts.createUser()
	u.RecoveryToken = "asdf"

	err := ts.db.Update(u)
	require.NoError(ts.T(), err)

	n, err := FindUserByRecoveryToken(ts.db, u.RecoveryToken)
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserWithRefreshToken() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u, GrantParams{})
	require.NoError(ts.T(), err)

	n, nr, err := FindUserWithRefreshToken(ts.db, r.Token) // using the original not hashed token
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), r.ID, nr.ID)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserWithRefreshTokenPrefixAttack() {
	// this scenario simulates an attack where the
	// attacker has gained access to the hashed token
	// values in the database and is trying to use one to
	// look up a user in the database. they should not be
	// able to do that.

	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u, GrantParams{})
	require.NoError(ts.T(), err)

	// using the hashed value
	_, _, err = FindUserWithRefreshToken(ts.db, r.HashedToken)
	require.Error(ts.T(), err)
	require.True(ts.T(), IsNotFoundError(err))

	// using the hashed value with a stripped prefix
	_, _, err = FindUserWithRefreshToken(ts.db, strings.TrimPrefix(r.HashedToken, "H:"))
	require.Error(ts.T(), err)
	require.True(ts.T(), IsNotFoundError(err))

	// using the hashed value with a lowercased prefix
	_, _, err = FindUserWithRefreshToken(ts.db, "h:"+strings.TrimPrefix(r.HashedToken, "H:"))
	require.Error(ts.T(), err)
	require.True(ts.T(), IsNotFoundError(err))
}

func (ts *UserTestSuite) TestIsDuplicatedEmail() {
	_ = ts.createUserWithEmail("david.calavera@netlify.com")

	e, err := IsDuplicatedEmail(ts.db, "david.calavera@netlify.com", "test")
	require.NoError(ts.T(), err)
	require.True(ts.T(), e, "expected email to be duplicated")

	e, err = IsDuplicatedEmail(ts.db, "davidcalavera@netlify.com", "test")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected email to not be duplicated")

	e, err = IsDuplicatedEmail(ts.db, "david@netlify.com", "test")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected same email to not be duplicated")

	e, err = IsDuplicatedEmail(ts.db, "david.calavera@netlify.com", "other-aud")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected same email to not be duplicated")
}

func (ts *UserTestSuite) createUser() *User {
	return ts.createUserWithEmail("david@netlify.com")
}

func (ts *UserTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser("", email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	err = ts.db.Create(user)
	require.NoError(ts.T(), err)

	return user
}

func (ts *UserTestSuite) TestRemoveUnconfirmedIdentities() {
	user, err := NewUser("+29382983298", "someone@example.com", "abcdefgh", "authenticated", nil)
	require.NoError(ts.T(), err)

	user.AppMetaData = map[string]interface{}{
		"provider":  "email",
		"providers": []string{"email", "phone", "twitter"},
	}

	require.NoError(ts.T(), ts.db.Create(user))

	idEmail, err := NewIdentity(user, "email", map[string]interface{}{
		"sub": "someone@example.com",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(idEmail))

	user.Identities = append(user.Identities, *idEmail)

	idPhone, err := NewIdentity(user, "phone", map[string]interface{}{
		"sub": "+29382983298",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(idPhone))

	user.Identities = append(user.Identities, *idPhone)

	idTwitter, err := NewIdentity(user, "twitter", map[string]interface{}{
		"sub": "test_twitter_user_id",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(idTwitter))

	user.Identities = append(user.Identities, *idTwitter)

	// reload the user
	require.NoError(ts.T(), ts.db.Load(user))

	require.False(ts.T(), user.IsConfirmed(), "user's email must not be confirmed")

	require.NoError(ts.T(), user.RemoveUnconfirmedIdentities(ts.db))

	require.Empty(ts.T(), user.EncryptedPassword, "password still remains in user")

	require.Len(ts.T(), user.Identities, 2, "only two identity must be remaining")
	require.Equal(ts.T(), idPhone.ID, user.Identities[0].ID, "remaining identity is not the expected one")
	require.Equal(ts.T(), idTwitter.ID, user.Identities[1].ID, "remaining identity is not the expected one")

	require.NotNil(ts.T(), user.AppMetaData)
	require.Equal(ts.T(), user.AppMetaData["provider"], "phone")
	require.Equal(ts.T(), user.AppMetaData["providers"], []string{"phone", "twitter"})
}
