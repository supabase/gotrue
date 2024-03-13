package api

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidCodeChallenge(t *testing.T) {
	cases := []struct {
		challenge     string
		isValid       bool
		expectedError error
	}{
		{
			challenge:     "invalid",
			isValid:       false,
			expectedError: badRequestError("code challenge has to be between %v and %v characters", MinCodeChallengeLength, MaxCodeChallengeLength),
		},
		{
			challenge:     "codechallengecontainsinvalidcharacterslike@$^&*",
			isValid:       false,
			expectedError: badRequestError("code challenge can only contain alphanumeric characters, hyphens, periods, underscores and tildes"),
		},
		{
			challenge:     "validchallengevalidchallengevalidchallengevalidchallenge",
			isValid:       true,
			expectedError: nil,
		},
	}

	for _, c := range cases {
		valid, err := isValidCodeChallenge(c.challenge)
		require.Equal(t, c.isValid, valid)
		require.Equal(t, c.expectedError, err)
	}
}

func TestIsValidPKCEParmas(t *testing.T) {
	cases := []struct {
		challengeMethod string
		challenge       string
		responseType    string
		expected        error
	}{
		{
			challengeMethod: "",
			challenge:       "",
			responseType:    "code",
			expected:        nil,
		},
		{
			challengeMethod: "test",
			challenge:       "testtesttesttesttesttesttesttesttesttesttesttesttesttesttesttest",
			responseType:    "code",
			expected:        nil,
		},
		{
			challengeMethod: "test",
			challenge:       "",
			responseType:    "code",
			expected:        badRequestError(InvalidPKCEParamsErrorMessage),
		},
		{
			challengeMethod: "",
			challenge:       "test",
			responseType:    "code",
			expected:        badRequestError(InvalidPKCEParamsErrorMessage),
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := validateCodeFlowParams(c.challengeMethod, c.challenge, c.responseType)
			require.Equal(t, c.expected, err)
		})
	}
}
