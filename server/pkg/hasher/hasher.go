// Package hasher provides password and SIP digest credential hashing helpers.
package hasher

import (
	"crypto/md5" // #nosec G501 -- MD5 is required by the SIP Digest protocol.
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HA1Hashes contains the HA1 digests used by SIP Digest authentication.
type HA1Hashes struct {
	MD5        string
	SHA256     string
	SHA512_256 string
}

func ComputeHA1(username, domain, password string) HA1Hashes {
	value := []byte(strings.Join([]string{username, domain, password}, ":"))
	md5Sum := md5.Sum(value)
	sha256Sum := sha256.Sum256(value)
	sha512Sum := sha512.Sum512_256(value)
	return HA1Hashes{MD5: hex.EncodeToString(md5Sum[:]), SHA256: hex.EncodeToString(sha256Sum[:]), SHA512_256: hex.EncodeToString(sha512Sum[:])}
}
func ComputeHA1MD5(username, domain, password string) string {
	return ComputeHA1(username, domain, password).MD5
}
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
func VerifyPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

// VerifyHA1 compares an expected digest without leaking mismatch position.
func VerifyHA1(expected, actual string) bool {
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
