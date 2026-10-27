package router

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func randomSessionKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(key)
}

func TestNewSessionStoreRejectsUnsafeKeys(t *testing.T) {
	for name, key := range map[string]string{
		"missing":            "",
		"short":              strings.Repeat("x", 31),
		"whitespace":         strings.Repeat(" \t\n", 16),
		"legacy":             "WhyJoiIsSoCute",
		"padded legacy":      "                    WhyJoiIsSoCute                    ",
		"sample":             "replace-with-a-long-random-secret",
		"padded sample":      " replace-with-a-long-random-secret\n",
		"development":        "dev-only-session-secret-change-in-production",
		"padded development": "\tdev-only-session-secret-change-in-production ",
		"short with padding": " " + strings.Repeat("x", 31),
	} {
		t.Run(name, func(t *testing.T) {
			store, err := NewSessionStore(key, false)
			if err == nil || store != nil {
				t.Fatal("unsafe key must fail without constructing a store")
			}
			if !strings.Contains(err.Error(), "security.session_secret") || !strings.Contains(err.Error(), "openssl rand -hex 32") {
				t.Fatalf("error must identify the setting and explain recovery: %v", err)
			}
			if key != "" && strings.Contains(err.Error(), key) {
				t.Fatal("error exposes the configured key")
			}
		})
	}
}

// Sign using the previous store constructor to exercise compatibility with real
// cookies, including forged administrator cookies made with published keys.
func signedSessionCookie(t *testing.T, key string) *http.Cookie {
	t.Helper()
	store := cookie.NewStore([]byte(key))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	session, err := store.Get(req, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user"] = uint(1)
	session.Values["authed"] = true
	response := httptest.NewRecorder()
	if err := store.Save(req, response, session); err != nil {
		t.Fatal(err)
	}
	return response.Result().Cookies()[0]
}

func TestSessionStorePreservesExistingValidSessions(t *testing.T) {
	key := randomSessionKey(t)
	for name, key := range map[string]string{
		"32 bytes":             key[:32],
		"64 bytes":             key,
		"unchanged whitespace": " " + key + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			store, err := NewSessionStore(key, false)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(signedSessionCookie(t, key))
			session, err := store.Get(req, "session")
			if err != nil || session.Values["authed"] != true || session.Values["user"] != uint(1) {
				t.Fatalf("existing valid session lost: session=%v error=%v", session, err)
			}
		})
	}
}

func TestSessionStoreRejectsForgedAndRotatedCookies(t *testing.T) {
	store, err := NewSessionStore(randomSessionKey(t), false)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(sessions.Sessions("session", store))
	for _, path := range []string{"/api/info", "/api/user"} {
		r.GET(path, authMiddleware, func(c *gin.Context) {
			t.Error("unauthorized request reached protected handler")
			c.Status(http.StatusNoContent)
		})
	}
	for name, key := range map[string]string{
		"no cookie":   "",
		"legacy":      "WhyJoiIsSoCute",
		"sample":      "replace-with-a-long-random-secret",
		"development": "dev-only-session-secret-change-in-production",
		"rotated":     randomSessionKey(t),
	} {
		t.Run(name, func(t *testing.T) {
			for _, path := range []string{"/api/info", "/api/user"} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				if key != "" {
					req.AddCookie(signedSessionCookie(t, key))
				}
				response := httptest.NewRecorder()
				r.ServeHTTP(response, req)
				if response.Code != http.StatusOK || response.Body.String() != `{"code":408,"message":"请先登录"}` {
					t.Fatalf("%s accepted unauthorized cookie: status=%d body=%s", path, response.Code, response.Body.String())
				}
			}
		})
	}
}

func TestSessionCookieOptions(t *testing.T) {
	for _, secure := range []bool{false, true} {
		store, err := NewSessionStore(randomSessionKey(t), secure)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		session, err := store.Get(req, "session")
		if err != nil {
			t.Fatal(err)
		}
		session.Values["authed"] = true
		response := httptest.NewRecorder()
		if err := store.Save(req, response, session); err != nil {
			t.Fatal(err)
		}
		cookie := response.Result().Cookies()[0]
		if cookie.Name != "session" || cookie.Path != "/" || cookie.Secure != secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("unexpected cookie options: %v", cookie)
		}
	}
}
