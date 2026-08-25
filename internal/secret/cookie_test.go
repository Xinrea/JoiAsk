package secret

import (
	"testing"

	"github.com/spf13/viper"
)

func TestCookieEncryptionRoundTrip(t *testing.T) {
	viper.Set("security.bilibili_cookie_key", "test-key-that-is-at-least-thirty-two-bytes")
	t.Cleanup(func() { viper.Set("security.bilibili_cookie_key", "") })
	encrypted, err := EncryptCookie("SESSDATA=sensitive")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "SESSDATA=sensitive" {
		t.Fatal("cookie was not encrypted")
	}
	plain, err := DecryptCookie(encrypted)
	if err != nil || plain != "SESSDATA=sensitive" {
		t.Fatalf("unexpected decrypted value %q, err=%v", plain, err)
	}
}
