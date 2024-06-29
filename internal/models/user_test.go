package models

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/supabase/auth/internal/conf"
	"github.com/supabase/auth/internal/crypto"
	"github.com/supabase/auth/internal/storage"
	"github.com/supabase/auth/internal/storage/test"
	"golang.org/x/crypto/bcrypt"
)

const modelsTestConfig = "../../hack/test.env"

func init() {
	crypto.PasswordHashCost = crypto.QuickHashCost
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
	tokenHash := "test_confirmation_token"
	require.NoError(ts.T(), CreateOneTimeToken(ts.db, u.ID, "relates_to not used", tokenHash, ConfirmationToken))

	n, err := FindUserByConfirmationToken(ts.db, tokenHash)
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
	tokenHash := "test_recovery_token"
	require.NoError(ts.T(), CreateOneTimeToken(ts.db, u.ID, "relates_to not used", tokenHash, RecoveryToken))

	n, err := FindUserByRecoveryToken(ts.db, tokenHash)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserWithRefreshToken() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u, GrantParams{})
	require.NoError(ts.T(), err)

	n, nr, s, err := FindUserWithRefreshToken(ts.db, r.Token, true /* forUpdate */)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), r.ID, nr.ID)
	require.Equal(ts.T(), u.ID, n.ID)
	require.NotNil(ts.T(), s)
	require.Equal(ts.T(), *r.SessionId, s.ID)
}

func (ts *UserTestSuite) TestIsDuplicatedEmail() {
	_ = ts.createUserWithEmail("david.calavera@netlify.com")

	e, err := IsDuplicatedEmail(ts.db, "david.calavera@netlify.com", "test", nil)
	require.NoError(ts.T(), err)
	require.NotNil(ts.T(), e, "expected email to be duplicated")

	e, err = IsDuplicatedEmail(ts.db, "davidcalavera@netlify.com", "test", nil)
	require.NoError(ts.T(), err)
	require.Nil(ts.T(), e, "expected email to not be duplicated", nil)

	e, err = IsDuplicatedEmail(ts.db, "david@netlify.com", "test", nil)
	require.NoError(ts.T(), err)
	require.Nil(ts.T(), e, "expected same email to not be duplicated", nil)

	e, err = IsDuplicatedEmail(ts.db, "david.calavera@netlify.com", "other-aud", nil)
	require.NoError(ts.T(), err)
	require.Nil(ts.T(), e, "expected same email to not be duplicated")
}

func (ts *UserTestSuite) createUser() *User {
	return ts.createUserWithEmail("david@netlify.com")
}

func (ts *UserTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser("", email, "secret", "test", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(user))

	identity, err := NewIdentity(user, "email", map[string]interface{}{
		"sub":   user.ID.String(),
		"email": email,
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(identity))

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

	idPhone, err := NewIdentity(user, "phone", map[string]interface{}{
		"sub": "+29382983298",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(idPhone))

	idTwitter, err := NewIdentity(user, "twitter", map[string]interface{}{
		"sub": "test_twitter_user_id",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(idTwitter))

	user.Identities = append(user.Identities, *idEmail, *idPhone, *idTwitter)

	// reload the user
	require.NoError(ts.T(), ts.db.Load(user))

	require.False(ts.T(), user.IsConfirmed(), "user's email must not be confirmed")

	require.NoError(ts.T(), user.RemoveUnconfirmedIdentities(ts.db, idTwitter))

	// reload the user to check that identities are deleted from the db too
	require.NoError(ts.T(), ts.db.Load(user))
	require.Empty(ts.T(), user.EncryptedPassword, "password still remains in user")

	require.Len(ts.T(), user.Identities, 1, "only one identity must be remaining")
	require.Equal(ts.T(), idTwitter.ID, user.Identities[0].ID, "remaining identity is not the expected one")

	require.NotNil(ts.T(), user.AppMetaData)
	require.Equal(ts.T(), user.AppMetaData["provider"], "twitter")
	require.Equal(ts.T(), user.AppMetaData["providers"], []string{"twitter"})
}

func (ts *UserTestSuite) TestConfirmEmailChange() {
	user, err := NewUser("", "test@example.com", "", "authenticated", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(user))

	identity, err := NewIdentity(user, "email", map[string]interface{}{
		"sub":   user.ID.String(),
		"email": "test@example.com",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(identity))

	user.EmailChange = "new@example.com"
	require.NoError(ts.T(), ts.db.UpdateOnly(user, "email_change"))

	require.NoError(ts.T(), user.ConfirmEmailChange(ts.db, 0))

	require.NoError(ts.T(), ts.db.Eager().Load(user))
	identity, err = FindIdentityByIdAndProvider(ts.db, user.ID.String(), "email")
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), user.Email, storage.NullString("new@example.com"))
	require.Equal(ts.T(), user.EmailChange, "")

	require.NotNil(ts.T(), identity.IdentityData)
	require.Equal(ts.T(), identity.IdentityData["email"], "new@example.com")
}

func (ts *UserTestSuite) TestConfirmPhoneChange() {
	user, err := NewUser("123456789", "", "", "authenticated", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(user))

	identity, err := NewIdentity(user, "phone", map[string]interface{}{
		"sub":   user.ID.String(),
		"phone": "123456789",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(identity))

	user.PhoneChange = "987654321"
	require.NoError(ts.T(), ts.db.UpdateOnly(user, "phone_change"))

	require.NoError(ts.T(), user.ConfirmPhoneChange(ts.db))

	require.NoError(ts.T(), ts.db.Eager().Load(user))
	identity, err = FindIdentityByIdAndProvider(ts.db, user.ID.String(), "phone")
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), user.Phone, storage.NullString("987654321"))
	require.Equal(ts.T(), user.PhoneChange, "")

	require.NotNil(ts.T(), identity.IdentityData)
	require.Equal(ts.T(), identity.IdentityData["phone"], "987654321")
}

func (ts *UserTestSuite) TestUpdateUserEmailSuccess() {
	userA, err := NewUser("", "foo@example.com", "", "authenticated", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(userA))

	primaryIdentity, err := NewIdentity(userA, "email", map[string]interface{}{
		"sub":   userA.ID.String(),
		"email": "foo@example.com",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(primaryIdentity))

	secondaryIdentity, err := NewIdentity(userA, "google", map[string]interface{}{
		"sub":   userA.ID.String(),
		"email": "bar@example.com",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(secondaryIdentity))

	// UpdateUserEmail should not do anything and the user's email should still use the primaryIdentity
	require.NoError(ts.T(), userA.UpdateUserEmailFromIdentities(ts.db))
	require.Equal(ts.T(), primaryIdentity.GetEmail(), userA.GetEmail())

	// remove primary identity
	require.NoError(ts.T(), ts.db.Destroy(primaryIdentity))

	// UpdateUserEmail should update the user to use the secondary identity's email
	require.NoError(ts.T(), userA.UpdateUserEmailFromIdentities(ts.db))
	require.Equal(ts.T(), secondaryIdentity.GetEmail(), userA.GetEmail())
}

func (ts *UserTestSuite) TestUpdateUserEmailFailure() {
	userA, err := NewUser("", "foo@example.com", "", "authenticated", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(userA))

	primaryIdentity, err := NewIdentity(userA, "email", map[string]interface{}{
		"sub":   userA.ID.String(),
		"email": "foo@example.com",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(primaryIdentity))

	secondaryIdentity, err := NewIdentity(userA, "google", map[string]interface{}{
		"sub":   userA.ID.String(),
		"email": "bar@example.com",
	})
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(secondaryIdentity))

	userB, err := NewUser("", "bar@example.com", "", "authenticated", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(userB))

	// remove primary identity
	require.NoError(ts.T(), ts.db.Destroy(primaryIdentity))

	// UpdateUserEmail should fail with the email unique constraint violation error
	//  since userB is using the secondary identity's email
	require.ErrorIs(ts.T(), userA.UpdateUserEmailFromIdentities(ts.db), UserEmailUniqueConflictError{})
	require.Equal(ts.T(), primaryIdentity.GetEmail(), userA.GetEmail())
}

func (ts *UserTestSuite) TestSetPasswordTooLong() {
	user, err := NewUser("", "", strings.Repeat("a", crypto.MaxPasswordLength), "", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.db.Create(user))

	err = user.SetPassword(ts.db.Context(), strings.Repeat("a", crypto.MaxPasswordLength+1), false, "", "")
	require.Error(ts.T(), err)

	err = user.SetPassword(ts.db.Context(), strings.Repeat("a", crypto.MaxPasswordLength), false, "", "")
	require.NoError(ts.T(), err)
}

func (ts *UserTestSuite) TestNewUserWithPasswordHashSuccess() {
	cases := []struct {
		desc string
		hash string
	}{
		{
			desc: "Valid bcrypt hash",
			hash: "$2y$10$SXEz2HeT8PUIGQXo9yeUIem8KzNxgG0d7o/.eGj2rj8KbRgAuRVlq",
		},
		{
			desc: "Valid argon2i hash",
			hash: "$argon2i$v=19$m=16,t=2,p=1$bGJRWThNOHJJTVBSdHl2dQ$NfEnUOuUpb7F2fQkgFUG4g",
		},
		{
			desc: "Valid argon2id hash",
			hash: "$argon2id$v=19$m=32,t=3,p=2$SFVpOWJ0eXhjRzVkdGN1RQ$RXnb8rh7LaDcn07xsssqqulZYXOM/EUCEFMVcAcyYVk",
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			u, err := NewUserWithPasswordHash("", "", c.hash, "", nil)
			require.NoError(ts.T(), err)
			require.NotNil(ts.T(), u)
		})
	}
}

func (ts *UserTestSuite) TestNewUserWithPasswordHashFailure() {
	cases := []struct {
		desc string
		hash string
	}{
		{
			desc: "Invalid argon2i hash",
			hash: "$argon2id$test",
		},
		{
			desc: "Invalid bcrypt hash",
			hash: "plaintest_password",
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			u, err := NewUserWithPasswordHash("", "", c.hash, "", nil)
			require.Error(ts.T(), err)
			require.Nil(ts.T(), u)
		})
	}
}

func (ts *UserTestSuite) TestAuthenticate() {
	// every case uses "test" as the password
	cases := []struct {
		desc             string
		hash             string
		expectedHashCost int
	}{
		{
			desc:             "Invalid bcrypt hash cost of 11",
			hash:             "$2y$11$4lH57PU7bGATpRcx93vIoObH3qDmft/pytbOzDG9/1WsyNmN5u4di",
			expectedHashCost: bcrypt.MinCost,
		},
		{
			desc:             "Valid bcrypt hash cost of 10",
			hash:             "$2y$10$va66S4MxFrH6G6L7BzYl0.QgcYgvSr/F92gc.3botlz7bG4p/g/1i",
			expectedHashCost: bcrypt.DefaultCost,
		},
	}

	for _, c := range cases {
		ts.Run(c.desc, func() {
			u, err := NewUserWithPasswordHash("", "", c.hash, "", nil)
			require.NoError(ts.T(), err)
			require.NoError(ts.T(), ts.db.Create(u))
			require.NotNil(ts.T(), u)

			isAuthenticated, _, err := u.Authenticate(context.Background(), ts.db, "test", nil, false, "")
			require.NoError(ts.T(), err)
			require.True(ts.T(), isAuthenticated)

			// check hash cost
			hashCost, err := bcrypt.Cost([]byte(*u.EncryptedPassword))
			require.NoError(ts.T(), err)
			require.Equal(ts.T(), c.expectedHashCost, hashCost)
		})
	}
}
