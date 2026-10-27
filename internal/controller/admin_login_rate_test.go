package controller

import (
	"encoding/json"
	"joiask-backend/internal/database"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAdminLoginRateTest(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousDB := database.DB
	path := filepath.Join(t.TempDir(), "admin-login-rate.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		database.DB = previousDB
		sqlDB.Close()
	})
	database.DB = db
	if err := db.AutoMigrate(&database.AdminLoginRateLimit{}); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	// This route intentionally has no user table or password checks: nonexistent
	// usernames and wrong passwords must consume exactly the same budget.
	r.POST("/login", func(c *gin.Context) {
		if !checkAdminLoginRate(c) {
			return
		}
		Fail(c, 401, "用户名或密码错误")
	})
	return r, path
}

func requestAdminLoginRate(t *testing.T, r *gin.Engine, ip string, status, code int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var response struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != status || response.Code != code {
		t.Fatalf("got HTTP %d, body %s; want HTTP %d, code %d", w.Code, w.Body.String(), status, code)
	}
	if response.Message == "" || string(response.Data) != "null" {
		t.Fatalf("invalid error envelope: %s", w.Body.String())
	}
	if code == 429 && w.Header().Get("Retry-After") != "60" {
		t.Fatalf("missing Retry-After: %v", w.Header())
	}
	return w
}

func TestAdminLoginRateRejectsEleventhThroughThirtiethAttempt(t *testing.T) {
	r, path := setupAdminLoginRateTest(t)
	for attempt := 1; attempt <= 30; attempt++ {
		status, code := http.StatusOK, 401
		if attempt > 10 {
			status, code = http.StatusTooManyRequests, 429
		}
		t.Run(strconv.Itoa(attempt), func(t *testing.T) {
			requestAdminLoginRate(t, r, "192.0.2.1", status, code)
		})
	}
	var bucket database.AdminLoginRateLimit
	if err := database.DB.First(&bucket).Error; err != nil {
		t.Fatal(err)
	}
	if bucket.Count != 10 || bucket.Key != hashVerificationToken("192.0.2.1") || len(bucket.Key) != 64 {
		t.Fatalf("unexpected bucket: %+v", bucket)
	}

	// A different backend connection shares the exhausted budget.
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	database.DB = db
	requestAdminLoginRate(t, r, "192.0.2.1", http.StatusTooManyRequests, 429)
	requestAdminLoginRate(t, r, "192.0.2.2", http.StatusOK, 401)
}

func TestAdminLoginRateWindowExpiryAndPruning(t *testing.T) {
	r, _ := setupAdminLoginRateTest(t)
	expired := time.Now().UTC().Add(-time.Minute - time.Second)
	for _, bucket := range []database.AdminLoginRateLimit{
		{Key: hashVerificationToken("192.0.2.1"), WindowStart: expired, Count: 10},
		{Key: hashVerificationToken("unused-old-ip"), WindowStart: expired, Count: 10},
		{Key: hashVerificationToken("active-other-ip"), WindowStart: time.Now().UTC(), Count: 10},
	} {
		if err := database.DB.Create(&bucket).Error; err != nil {
			t.Fatal(err)
		}
	}
	for attempt := 1; attempt <= 10; attempt++ {
		requestAdminLoginRate(t, r, "192.0.2.1", http.StatusOK, 401)
	}
	requestAdminLoginRate(t, r, "192.0.2.1", http.StatusTooManyRequests, 429)
	var buckets []database.AdminLoginRateLimit
	if err := database.DB.Find(&buckets).Error; err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expired unused bucket was not pruned: %+v", buckets)
	}
	for _, bucket := range buckets {
		if !bucket.WindowStart.After(expired) || bucket.Count != 10 {
			t.Fatalf("unexpected bucket after expiry: %+v", bucket)
		}
	}

	// Cleanup also commits on rejected attempts.
	stale := database.AdminLoginRateLimit{Key: hashVerificationToken("another-old-ip"), WindowStart: expired, Count: 1}
	if err := database.DB.Create(&stale).Error; err != nil {
		t.Fatal(err)
	}
	requestAdminLoginRate(t, r, "192.0.2.1", http.StatusTooManyRequests, 429)
	var count int64
	if err := database.DB.Model(&database.AdminLoginRateLimit{}).Where("`key` = ?", stale.Key).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("rejected attempt did not prune expired bucket")
	}
}

func TestAdminLoginRateDatabaseFailure(t *testing.T) {
	for _, failure := range []string{"missing table", "closed connection"} {
		t.Run(failure, func(t *testing.T) {
			r, _ := setupAdminLoginRateTest(t)
			if failure == "missing table" {
				if err := database.DB.Migrator().DropTable(&database.AdminLoginRateLimit{}); err != nil {
					t.Fatal(err)
				}
			} else {
				sqlDB, err := database.DB.DB()
				if err != nil {
					t.Fatal(err)
				}
				if err := sqlDB.Close(); err != nil {
					t.Fatal(err)
				}
			}
			// Preserve the normal Fail envelope/HTTP status, without proceeding
			// to the route's simulated password check (which returns code 401).
			w := requestAdminLoginRate(t, r, "192.0.2.1", http.StatusOK, 500)
			if w.Header().Get("Retry-After") != "" {
				t.Fatal("database error must not be reported as throttling")
			}
		})
	}
}
