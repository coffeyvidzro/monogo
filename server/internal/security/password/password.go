package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	algorithm          = "argon2id"
	defaultMemory      = uint32(64 * 1024)
	defaultIterations  = uint32(3)
	defaultParallelism = uint8(4)
	defaultSaltLength  = 16
	defaultKeyLength   = uint32(32)
	maxMemory          = uint32(256 * 1024)
	maxIterations      = uint32(10)
	maxParallelism     = uint8(16)
	maxSaltLength      = 64
	maxKeyLength       = uint32(128)
)

type parameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	keyLength   uint32
}

// Hash returns an Argon2id encoded password hash using the current parameters.
func Hash(value string) (string, error) {
	salt := make([]byte, defaultSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(value),
		salt,
		defaultIterations,
		defaultMemory,
		defaultParallelism,
		defaultKeyLength,
	)

	return fmt.Sprintf(
		"%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		algorithm,
		argon2.Version,
		defaultMemory,
		defaultIterations,
		defaultParallelism,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	), nil
}

// Verify checks a password against an encoded Argon2id hash. The parameters
// embedded in the encoded value are authoritative so hashes remain verifiable
// when the defaults change in a future release.
func Verify(value, encoded string) bool {
	params, salt, expected, err := decode(encoded)
	if err != nil {
		return false
	}

	actual := argon2.IDKey(
		[]byte(value),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		params.keyLength,
	)

	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// NeedsRehash reports whether an encoded password hash uses parameters other
// than the current defaults.
func NeedsRehash(encoded string) bool {
	params, salt, _, err := decode(encoded)
	if err != nil {
		return true
	}

	return params.memory != defaultMemory ||
		params.iterations != defaultIterations ||
		params.parallelism != defaultParallelism ||
		params.keyLength != defaultKeyLength ||
		len(salt) != defaultSaltLength
}

func decode(encoded string) (parameters, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != algorithm {
		return parameters{}, nil, nil, fmt.Errorf("invalid password hash format")
	}

	version, ok := strings.CutPrefix(parts[1], "v=")
	if !ok {
		return parameters{}, nil, nil, fmt.Errorf("invalid password hash version")
	}
	parsedVersion, err := strconv.ParseUint(version, 10, 8)
	if err != nil || int(parsedVersion) != argon2.Version {
		return parameters{}, nil, nil, fmt.Errorf("unsupported password hash version")
	}

	params, err := parseParameters(parts[2])
	if err != nil {
		return parameters{}, nil, nil, err
	}

	salt, err := hex.DecodeString(parts[3])
	if err != nil || len(salt) == 0 {
		return parameters{}, nil, nil, fmt.Errorf("invalid password hash salt")
	}

	expected, err := hex.DecodeString(parts[4])
	if err != nil || len(expected) == 0 || len(expected) > int(maxKeyLength) {
		return parameters{}, nil, nil, fmt.Errorf("invalid password hash digest")
	}
	params.keyLength = uint32(len(expected))
	if params.memory > maxMemory || params.iterations > maxIterations || params.parallelism > maxParallelism || len(salt) > maxSaltLength {
		return parameters{}, nil, nil, fmt.Errorf("password hash parameters exceed verification limits")
	}

	return params, salt, expected, nil
}

func parseParameters(encoded string) (parameters, error) {
	values := strings.Split(encoded, ",")
	if len(values) != 3 {
		return parameters{}, fmt.Errorf("invalid password hash parameters")
	}

	memory, err := parseUintParameter(values[0], "m=", 32)
	if err != nil || memory == 0 {
		return parameters{}, fmt.Errorf("invalid password hash memory parameter")
	}
	iterations, err := parseUintParameter(values[1], "t=", 32)
	if err != nil || iterations == 0 {
		return parameters{}, fmt.Errorf("invalid password hash iterations parameter")
	}
	parallelism, err := parseUintParameter(values[2], "p=", 8)
	if err != nil || parallelism == 0 {
		return parameters{}, fmt.Errorf("invalid password hash parallelism parameter")
	}

	return parameters{
		memory:      uint32(memory),
		iterations:  uint32(iterations),
		parallelism: uint8(parallelism),
	}, nil
}

func parseUintParameter(value, prefix string, bitSize int) (uint64, error) {
	value, ok := strings.CutPrefix(value, prefix)
	if !ok {
		return 0, fmt.Errorf("missing parameter prefix %q", prefix)
	}
	return strconv.ParseUint(value, 10, bitSize)
}
