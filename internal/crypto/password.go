package crypto

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/supabase/auth/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

type HashCost = int

const (
	// DefaultHashCost represents the default
	// hashing cost for any hashing algorithm.
	DefaultHashCost HashCost = iota

	// QuickHashCosts represents the quickest
	// hashing cost for any hashing algorithm,
	// useful for tests only.
	QuickHashCost HashCost = iota

	// BCrypt hashed passwords have a 72 character limit
	MaxPasswordLength = 72

	Argon2Prefix = "$argon2"
)

// PasswordHashCost is the current pasword hashing cost
// for all new hashes generated with
// GenerateHashFromPassword.
var PasswordHashCost = DefaultHashCost

var (
	generateFromPasswordSubmittedCounter = observability.ObtainMetricCounter("gotrue_generate_from_password_submitted", "Number of submitted GenerateFromPassword hashing attempts")
	generateFromPasswordCompletedCounter = observability.ObtainMetricCounter("gotrue_generate_from_password_completed", "Number of completed GenerateFromPassword hashing attempts")
)

var (
	compareHashAndPasswordSubmittedCounter = observability.ObtainMetricCounter("gotrue_compare_hash_and_password_submitted", "Number of submitted CompareHashAndPassword hashing attempts")
	compareHashAndPasswordCompletedCounter = observability.ObtainMetricCounter("gotrue_compare_hash_and_password_completed", "Number of completed CompareHashAndPassword hashing attempts")
)

var ErrArgon2MismatchedHashAndPassword = errors.New("crypto: argon2 hash and password mismatch")

// argon2HashRegexp https://github.com/P-H-C/phc-string-format/blob/master/phc-sf-spec.md#argon2-encoding
var argon2HashRegexp = regexp.MustCompile("^[$](?P<alg>argon2(d|i|id))[$]v=(?P<v>(16|19))[$]m=(?P<m>[0-9]+),t=(?P<t>[0-9]+),p=(?P<p>[0-9]+)(,keyid=(?P<keyid>[^,]+))?(,data=(?P<data>[^$]+))?[$](?P<salt>[^$]+)[$](?P<hash>.+)$")

type Argon2HashInput struct {
	alg     string
	v       string
	memory  uint64
	time    uint64
	threads uint64
	keyid   string
	data    string
	salt    []byte
	rawHash []byte
}

func ParseArgon2Hash(hash string) (*Argon2HashInput, error) {
	submatch := argon2HashRegexp.FindStringSubmatchIndex(hash)

	if submatch == nil {
		return nil, errors.New("crypto: incorrect argon2 hash format")
	}

	alg := string(argon2HashRegexp.ExpandString(nil, "$alg", hash, submatch))
	v := string(argon2HashRegexp.ExpandString(nil, "$v", hash, submatch))
	m := string(argon2HashRegexp.ExpandString(nil, "$m", hash, submatch))
	t := string(argon2HashRegexp.ExpandString(nil, "$t", hash, submatch))
	p := string(argon2HashRegexp.ExpandString(nil, "$p", hash, submatch))
	keyid := string(argon2HashRegexp.ExpandString(nil, "$keyid", hash, submatch))
	data := string(argon2HashRegexp.ExpandString(nil, "$data", hash, submatch))
	saltB64 := string(argon2HashRegexp.ExpandString(nil, "$salt", hash, submatch))
	hashB64 := string(argon2HashRegexp.ExpandString(nil, "$hash", hash, submatch))

	if alg != "argon2i" && alg != "argon2id" {
		return nil, fmt.Errorf("crypto: argon2 hash uses unsupported algorithm %q only argon2i and argon2id supported", alg)
	}

	if v != "19" {
		return nil, fmt.Errorf("crypto: argon2 hash uses unsupported version %q only %d is supported", v, argon2.Version)
	}

	if data != "" {
		return nil, fmt.Errorf("crypto: argon2 hashes with the data parameter not supported")
	}

	if keyid != "" {
		return nil, fmt.Errorf("crypto: argon2 hashes with the keyid parameter not supported")
	}

	memory, err := strconv.ParseUint(m, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("crypto: argon2 hash has invalid m parameter %q %w", m, err)
	}

	time, err := strconv.ParseUint(t, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("crypto: argon2 hash has invalid t parameter %q %w", t, err)
	}

	threads, err := strconv.ParseUint(p, 10, 8)
	if err != nil {
		return nil, fmt.Errorf("crypto: argon2 hash has invalid p parameter %q %w", p, err)
	}

	rawHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return nil, fmt.Errorf("crypto: argon2 hash has invalid base64 in the hash section %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("crypto: argon2 hash has invalid base64 in the salt section %w", err)
	}

	input := Argon2HashInput{alg, v, memory, time, threads, keyid, data, salt, rawHash}

	return &input, nil
}

func compareHashAndPasswordArgon2(ctx context.Context, hash, password string) error {
	input, err := ParseArgon2Hash(hash)
	if err != nil {
		return err
	}

	attributes := []attribute.KeyValue{
		attribute.String("alg", input.alg),
		attribute.String("v", input.v),
		attribute.Int64("m", int64(input.memory)),
		attribute.Int64("t", int64(input.time)),
		attribute.Int("p", int(input.threads)),
		attribute.Int("len", len(input.rawHash)),
	}

	var match bool
	var derivedKey []byte
	compareHashAndPasswordSubmittedCounter.Add(ctx, 1, metric.WithAttributes(attributes...))
	defer func() {
		attributes = append(attributes, attribute.Bool(
			"match",
			match,
		))

		compareHashAndPasswordCompletedCounter.Add(ctx, 1, metric.WithAttributes(attributes...))
	}()

	switch input.alg {
	case "argon2i":
		derivedKey = argon2.Key([]byte(password), input.salt, uint32(input.time), uint32(input.memory)*1024, uint8(input.threads), uint32(len(input.rawHash)))

	case "argon2id":
		derivedKey = argon2.IDKey([]byte(password), input.salt, uint32(input.time), uint32(input.memory)*1024, uint8(input.threads), uint32(len(input.rawHash)))
	}

	match = subtle.ConstantTimeCompare(derivedKey, input.rawHash) == 0

	if !match {
		return ErrArgon2MismatchedHashAndPassword
	}

	return nil
}

// CompareHashAndPassword compares the hash and
// password, returns nil if equal otherwise an error. Context can be used to
// cancel the hashing if the algorithm supports it.
func CompareHashAndPassword(ctx context.Context, hash, password string) error {
	if strings.HasPrefix(hash, Argon2Prefix) {
		return compareHashAndPasswordArgon2(ctx, hash, password)
	}

	// assume bcrypt
	hashCost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return err
	}

	attributes := []attribute.KeyValue{
		attribute.String("alg", "bcrypt"),
		attribute.Int("bcrypt_cost", hashCost),
	}

	compareHashAndPasswordSubmittedCounter.Add(ctx, 1, metric.WithAttributes(attributes...))
	defer func() {
		attributes = append(attributes, attribute.Bool(
			"match",
			!errors.Is(err, bcrypt.ErrMismatchedHashAndPassword),
		))

		compareHashAndPasswordCompletedCounter.Add(ctx, 1, metric.WithAttributes(attributes...))
	}()

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err
}

// GenerateFromPassword generates a password hash from a
// password, using PasswordHashCost. Context can be used to cancel the hashing
// if the algorithm supports it.
func GenerateFromPassword(ctx context.Context, password string) (string, error) {
	var hashCost int

	if len(password) > MaxPasswordLength {
		return "", fmt.Errorf("password cannot be longer than %d characters", MaxPasswordLength)
	}

	switch PasswordHashCost {
	case QuickHashCost:
		hashCost = bcrypt.MinCost

	default:
		hashCost = bcrypt.DefaultCost
	}

	attributes := []attribute.KeyValue{
		attribute.String("alg", "bcrypt"),
		attribute.Int("bcrypt_cost", hashCost),
	}

	generateFromPasswordSubmittedCounter.Add(ctx, 1, metric.WithAttributes(attributes...))
	defer generateFromPasswordCompletedCounter.Add(ctx, 1, metric.WithAttributes(attributes...))

	hash, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
