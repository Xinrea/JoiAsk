// Package adminpassword supports bcrypt credentials and the one-way migration of
// legacy administrator MD5 credentials. Callers always supply the raw password.
package adminpassword

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"joiask-backend/pkg/util"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var ErrEmptyPassword = errors.New("密码不能为空")

// The versioned prefix distinguishes bcrypt(hex(SHA256(password))) from
// bcrypt(password). Hex encoding keeps the bcrypt input at 64 bytes, without
// embedded NULs. Only passwords exceeding bcrypt's 72-byte limit use this format.
const sha256Prefix = "$bcrypt-sha256$v1$"

func Hash(password string) (string, error) {
	if len(password) == 0 {
		return "", ErrEmptyPassword
	}
	prefix := ""
	if len(password) > 72 {
		password = sha256Hex(password)
		prefix = sha256Prefix
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return prefix + string(hash), nil
}

// Verify reports whether the raw password matches and whether a matching legacy
// credential needs migration. Never compare a submitted hash directly to stored.
func Verify(stored, password string) (matches, needsMigration bool) {
	if len(password) == 0 {
		return false, false
	}
	if len(stored) == 32 {
		if _, err := hex.DecodeString(stored); err == nil {
			matches = subtle.ConstantTimeCompare([]byte(strings.ToLower(stored)), []byte(util.Md5v(password))) == 1
			return matches, matches
		}
	}
	if strings.HasPrefix(stored, sha256Prefix) {
		stored = strings.TrimPrefix(stored, sha256Prefix)
		password = sha256Hex(password)
	} else if len(password) > 72 {
		// CompareHashAndPassword otherwise accepts overlong inputs by ignoring
		// everything past byte 72. Untagged hashes must never use that behavior.
		return false, false
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil, false
}

func sha256Hex(password string) string {
	digest := sha256.Sum256([]byte(password))
	return hex.EncodeToString(digest[:])
}
