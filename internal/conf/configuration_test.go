package conf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	defer os.Clearenv()
	os.Exit(m.Run())
}

func TestGlobal(t *testing.T) {
	os.Setenv("GOTRUE_SITE_URL", "http://localhost:8080")
	os.Setenv("GOTRUE_DB_DRIVER", "postgres")
	os.Setenv("GOTRUE_DB_DATABASE_URL", "fake")
	os.Setenv("GOTRUE_OPERATOR_TOKEN", "token")
	os.Setenv("GOTRUE_API_REQUEST_ID_HEADER", "X-Request-ID")
	os.Setenv("GOTRUE_JWT_SECRET", "secret")
	os.Setenv("API_EXTERNAL_URL", "http://localhost:9999")
	os.Setenv("GOTRUE_HOOK_MFA_VERIFICATION_ATTEMPT_URI", "pg-functions://postgres/auth/count_failed_attempts")
	os.Setenv("GOTRUE_HOOK_SEND_SMS_SECRETS", "v1,whsec_aWxpa2VzdXBhYmFzZXZlcnltdWNoYW5kaWhvcGV5b3Vkb3Rvbw==")
	gc, err := LoadGlobal("")
	require.NoError(t, err)
	require.NotNil(t, gc)
	assert.Equal(t, "X-Request-ID", gc.API.RequestIDHeader)
	assert.Equal(t, "pg-functions://postgres/auth/count_failed_attempts", gc.Hook.MFAVerificationAttempt.URI)
}

func TestPasswordRequiredCharactersDecode(t *testing.T) {
	examples := []struct {
		Value  string
		Result []string
	}{
		{
			Value: "a:b:c",
			Result: []string{
				"a",
				"b",
				"c",
			},
		},
		{
			Value: "a\\:b:c",
			Result: []string{
				"a:b",
				"c",
			},
		},
		{
			Value: "a:b\\:c",
			Result: []string{
				"a",
				"b:c",
			},
		},
		{
			Value: "\\:a:b:c",
			Result: []string{
				":a",
				"b",
				"c",
			},
		},
		{
			Value: "a:b:c\\:",
			Result: []string{
				"a",
				"b",
				"c:",
			},
		},
		{
			Value: "::\\::",
			Result: []string{
				":",
			},
		},
		{
			Value:  "",
			Result: nil,
		},
		{
			Value: " ",
			Result: []string{
				" ",
			},
		},
	}

	for i, example := range examples {
		var into PasswordRequiredCharacters
		require.NoError(t, into.Decode(example.Value), "Example %d failed with error", i)

		require.Equal(t, []string(into), example.Result, "Example %d got unexpected result", i)
	}
}

func TestHTTPHookSecretsDecode(t *testing.T) {
	examples := []struct {
		Value  string
		Result []string
	}{
		{
			Value:  "v1,whsec_secret1|v1a,whpk_secrets:whsk_secret2|v1,whsec_secret3",
			Result: []string{"v1,whsec_secret1", "v1a,whpk_secrets:whsk_secret2", "v1,whsec_secret3"},
		},
		{
			Value:  "v1,whsec_singlesecret",
			Result: []string{"v1,whsec_singlesecret"},
		},
		{
			Value:  " ",
			Result: []string{" "},
		},
		{
			Value:  "",
			Result: nil,
		},
		{
			Value: "|a|b|c",
			Result: []string{
				"a",
				"b",
				"c",
			},
		},
		{
			Value:  "||||",
			Result: nil,
		},
		{
			Value:  "::",
			Result: []string{"::"},
		},
		{
			Value:  "secret1::secret3",
			Result: []string{"secret1::secret3"},
		},
	}

	for i, example := range examples {
		var into HTTPHookSecrets

		require.NoError(t, into.Decode(example.Value), "Example %d failed with error", i)
		require.Equal(t, []string(into), example.Result, "Example %d got unexpected result", i)
	}
}

func TestValidateExtensibilityPointURI(t *testing.T) {
	cases := []struct {
		desc        string
		uri         string
		expectError bool
	}{
		// Positive test cases
		{desc: "Valid HTTPS URI", uri: "https://asdfgggqqwwerty.website.co/functions/v1/custom-sms-sender", expectError: false},
		{desc: "Valid HTTPS URI", uri: "HTTPS://www.asdfgggqqwwerty.website.co/functions/v1/custom-sms-sender", expectError: false},
		{desc: "Valid Postgres URI", uri: "pg-functions://postgres/auth/verification_hook_reject", expectError: false},
		{desc: "Another Valid URI", uri: "pg-functions://postgres/user_management/add_user", expectError: false},
		{desc: "Another Valid URI", uri: "pg-functions://postgres/MySpeCial/FUNCTION_THAT_YELLS_AT_YOU", expectError: false},
		{desc: "Valid HTTP URI", uri: "http://localhost/functions/v1/custom-sms-sender", expectError: false},
		{desc: "Valid localhost URI with kong alias", uri: "http://kong:8000/functions/v1/custom-sms-sender", expectError: false},
		{desc: "Valid localhost URI with edge_runtime", uri: "http://edge_runtime:54321/functions/v1/custom-sms-sender", expectError: false},

		// Negative test cases
		{desc: "Invalid HTTP URI", uri: "http://asdfgggg.website.co/functions/v1/custom-sms-sender", expectError: true},
		{desc: "Invalid HTTPS URI (HTTP)", uri: "http://asdfgggqqwwerty.supabase.co/functions/v1/custom-sms-sender", expectError: true},
		{desc: "Invalid Schema Name", uri: "pg-functions://postgres/123auth/verification_hook_reject", expectError: true},
		{desc: "Invalid Function Name", uri: "pg-functions://postgres/auth/123verification_hook_reject", expectError: true},
		{desc: "Insufficient Path Parts", uri: "pg-functions://postgres/auth", expectError: true},
	}

	for _, tc := range cases {
		ep := ExtensibilityPointConfiguration{URI: tc.uri}
		err := ep.ValidateExtensibilityPoint()
		if tc.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestValidateExtensibilityPointSecrets(t *testing.T) {
	validHTTPSURI := "https://asdfgggqqwwerty.website.co/functions/v1/custom-sms-sender"
	cases := []struct {
		desc        string
		secret      []string
		expectError bool
	}{
		// Positive test cases
		{desc: "Valid Symmetric Secret", secret: []string{"v1,whsec_NDYzODhlNTY0ZGI1OWZjYTU2NjMwN2FhYzM3YzBkMWQ0NzVjNWRkNTJmZDU0MGNhYTAzMjVjNjQzMzE3Mjk2Zg====="}, expectError: false},
		{desc: "Valid Asymmetric Secret", secret: []string{"v1a,whpk_NDYzODhlNTY0ZGI1OWZjYTU2NjMwN2FhYzM3YzBkMWQ0NzVjNWRkNTJmZDU0MGNhYTAzMjVjNjQzMzE3Mjk2Zg==:whsk_abc889a6b1160015025064f108a48d6aba1c7c95fa8e304b4d225e8ae0121511"}, expectError: false},
		{desc: "Valid Mix of Symmetric and asymmetric Secret", secret: []string{"v1,whsec_2b49264c90fd15db3bb0e05f4e1547b9c183eb06d585be8a", "v1a,whpk_46388e564db59fca566307aac37c0d1d475c5dd52fd540caa0325c643317296f:whsk_YWJjODg5YTZiMTE2MDAxNTAyNTA2NGYxMDhhNDhkNmFiYTFjN2M5NWZhOGUzMDRiNGQyMjVlOGFlMDEyMTUxMSI="}, expectError: false},

		// Negative test cases
		{desc: "Invalid Asymmetric Secret", secret: []string{"v1a,john:jill", "jill"}, expectError: true},
		{desc: "Invalid Symmetric Secret", secret: []string{"tommy"}, expectError: true},
	}
	for _, tc := range cases {
		ep := ExtensibilityPointConfiguration{URI: validHTTPSURI, HTTPHookSecrets: tc.secret}
		err := ep.ValidateExtensibilityPoint()
		if tc.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

	}

}
