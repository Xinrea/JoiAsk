package router

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
)

// NewSessionStore validates the signing key before creating the cookie store.
// Call it before initializing the database or starting background workers.
func NewSessionStore(key string, secure bool) (sessions.Store, error) {
	if err := validateSessionSecret(key); err != nil {
		return nil, err
	}
	// Preserve the configured bytes so existing sessions signed with valid keys
	// remain valid. Trimming below is only for validation.
	store := cookie.NewStore([]byte(key))
	store.Options(sessions.Options{
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return store, nil
}

// Length alone cannot establish randomness; operators must provide their own key.
func validateSessionSecret(value string) error {
	const minimumBytes = 32
	secret := strings.TrimSpace(value)
	if len(secret) < minimumBytes {
		return errors.New("security.session_secret must be at least 32 bytes; generate a random key with: openssl rand -hex 32")
	}
	switch secret {
	case "WhyJoiIsSoCute", "replace-with-a-long-random-secret", "dev-only-session-secret-change-in-production":
		return errors.New("security.session_secret must not be a published example or legacy key; generate a random key with: openssl rand -hex 32")
	}
	return nil
}
