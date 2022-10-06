package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"

	"github.com/pkg/errors"
)

// SecureToken is an object that represents a unique randomly generated string
// that can be sent to a client and/or stored in a database for lookup only.
type SecureToken struct {
	Original string `json:"-"`
	Hashed   string `json:"-"`
}

// SecureToken creates a new random token
func GenerateSecureToken() SecureToken {
	bytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		panic(err.Error()) // rand should never fail
	}

	original := base64.RawURLEncoding.EncodeToString(bytes)

	return SecureToken{
		Original: original,
		Hashed:   HashSHA224Base64(original),
	}
}

// HashSHA224 hashes the provided string with SHA256/224 and returns it as
// Base64 URL encoded. SHA256/224 is a good hashing function as it's shorter
// than SHA256 but also is not succeptible to a length extension attack.
func HashSHA224Base64(str string) string {
	bytes := sha256.Sum224([]byte(str))

	return base64.RawURLEncoding.EncodeToString(bytes[:])
}

// HashSHA224 hashes the provided string with SHA256/224 and returns it as
// hex encoded. SHA256/224 is a good hashing function as it's shorter
// than SHA256 but also is not succeptible to a length extension attack.
func HashSHA224Hex(str string) string {
	bytes := sha256.Sum224([]byte(str))

	return hex.EncodeToString(bytes[:])
}

// GenerateOtp generates a random n digit otp
func GenerateOtp(digits int) (string, error) {
	upper := math.Pow10(digits)
	val, err := rand.Int(rand.Reader, big.NewInt(int64(upper)))
	if err != nil {
		return "", errors.WithMessage(err, "Error generating otp")
	}
	// adds a variable zero-padding to the left to ensure otp is uniformly random
	expr := "%0" + strconv.Itoa(digits) + "v"
	otp := fmt.Sprintf(expr, val.String())
	return otp, nil
}
