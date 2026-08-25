package verification

import (
	"context"
	"io"
	"joiask-backend/internal/bilibili"
	"joiask-backend/internal/database"
	"joiask-backend/internal/secret"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestWorkerMatchesWindowBeforeExpiringRequests(t *testing.T) {
	now := time.Unix(1700000181, 0).UTC()
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := `{"code":0,"message":"0","data":{"list":[{"mid":101,"mtime":1700000179,"uname":"matched","face":"https://example.test/avatar.jpg"},{"mid":202,"mtime":1699999999,"uname":"old","face":""}]}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	var err error
	database.DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "worker.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.AutoMigrate(&database.BilibiliVerificationAccount{}, &database.BilibiliVerificationRequest{}); err != nil {
		t.Fatal(err)
	}
	viper.Set("security.bilibili_cookie_key", "test-key-that-is-at-least-thirty-two-bytes")
	t.Cleanup(func() { viper.Set("security.bilibili_cookie_key", "") })
	encrypted, err := secret.EncryptCookie("SESSDATA=test")
	if err != nil {
		t.Fatal(err)
	}
	database.DB.Create(&database.BilibiliVerificationAccount{BilibiliUID: 999, EncryptedCookie: encrypted})
	requests := []database.BilibiliVerificationRequest{
		{BilibiliUID: 101, CredentialHash: "a", Status: database.VerificationPending, RequestedAt: now.Add(-3 * time.Minute), ExpiresAt: now.Add(-time.Second)},
		{BilibiliUID: 202, CredentialHash: "b", Status: database.VerificationPending, RequestedAt: now.Add(-3 * time.Minute), ExpiresAt: now.Add(-time.Second)},
	}
	for i := range requests {
		if err := database.DB.Create(&requests[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	worker := NewWorker(&bilibili.Client{BaseURL: "https://example.test", HTTPClient: httpClient})
	worker.now = func() time.Time { return now }
	worker.RunOnce(context.Background())

	var matched, expired database.BilibiliVerificationRequest
	database.DB.First(&matched, requests[0].ID)
	database.DB.First(&expired, requests[1].ID)
	if matched.Status != database.VerificationVerified || matched.BilibiliName != "matched" || matched.FollowedAt == nil {
		t.Fatalf("request was not verified: %+v", matched)
	}
	if expired.Status != database.VerificationExpired {
		t.Fatalf("old request was not expired: %+v", expired)
	}
	worker.RunOnce(context.Background())
	database.DB.First(&matched, requests[0].ID)
	if matched.Status != database.VerificationVerified {
		t.Fatalf("repeated scan changed verified request: %+v", matched)
	}
}
