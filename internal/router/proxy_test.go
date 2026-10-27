package router

import (
	"fmt"
	"joiask-backend/internal/controller"
	"joiask-backend/internal/database"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTrustedProxyClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, remote, forwarded, want string
		proxies                       []string
	}{
		{"direct public", "192.0.2.10:123", "198.51.100.1", "192.0.2.10", nil},
		{"local Caddy", "127.0.0.1:123", "198.51.100.1, 192.0.2.10", "192.0.2.10", nil},
		{"outer Nginx", "127.0.0.1:123", "198.51.100.1, 192.0.2.10, 172.17.0.1", "192.0.2.10", nil},
		{"container Caddy", "172.17.0.2:123", "198.51.100.1, 192.0.2.10, 172.17.0.1", "192.0.2.10", nil},
		{"IPv6 client", "[::1]:123", "198.51.100.1, 2001:db8::1", "2001:db8::1", nil},
		{"explicit proxy", "192.0.2.20:123", "198.51.100.1, 192.0.2.10", "192.0.2.10", []string{"192.0.2.20"}},
		{"untrusted private peer", "172.17.0.2:123", "198.51.100.1", "172.17.0.2", []string{"127.0.0.1"}},
		{"no proxies", "127.0.0.1:123", "198.51.100.1", "127.0.0.1", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.proxies == nil {
				viper.Set("security.trusted_proxies", nil)
			} else {
				viper.Set("security.trusted_proxies", tc.proxies)
			}
			t.Cleanup(func() { viper.Set("security.trusted_proxies", nil) })
			r := gin.New()
			if err := configureTrustedProxies(r); err != nil {
				t.Fatal(err)
			}
			r.GET("/", func(c *gin.Context) { c.String(200, c.ClientIP()) })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			req.Header.Set("X-Forwarded-For", tc.forwarded)
			req.Header.Set("X-Real-IP", "198.51.100.99")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Body.String() != tc.want {
				t.Fatalf("client IP = %q, want %q", w.Body.String(), tc.want)
			}
		})
	}
}

func TestTrustedProxyRejectsInvalidConfiguration(t *testing.T) {
	viper.Set("security.trusted_proxies", []string{"not-an-ip"})
	t.Cleanup(func() { viper.Set("security.trusted_proxies", nil) })
	if err := configureTrustedProxies(gin.New()); err == nil {
		t.Fatal("invalid trusted proxy network accepted")
	}
}

func TestAdminLoginRateCannotBeBypassedWithForwardedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	viper.Set("security.trusted_proxies", nil)
	previous := database.DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "proxy-login.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		database.DB = previous
		sqlDB.Close()
	})
	if err := db.AutoMigrate(&database.Admin{}, &database.AdminLoginRateLimit{}); err != nil {
		t.Fatal(err)
	}
	for _, remote := range []string{"192.0.2.10:123", "127.0.0.1:123", "172.17.0.2:123"} {
		t.Run(remote, func(t *testing.T) {
			if err := db.Where("1 = 1").Delete(&database.AdminLoginRateLimit{}).Error; err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			if err := configureTrustedProxies(r); err != nil {
				t.Fatal(err)
			}
			r.POST("/login", (&controller.UserController{}).Login)
			for attempt := 1; attempt <= 30; attempt++ {
				req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"unknown","password":"wrong"}`))
				req.RemoteAddr = remote
				req.Header.Set("Content-Type", "application/json")
				// The attacker controls the leftmost entry; Nginx appends the real
				// client and Caddy may append a private outer-proxy address.
				req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d, 192.0.2.10, 172.17.0.1", attempt))
				req.Header.Set("X-Real-IP", fmt.Sprintf("198.51.100.%d", attempt))
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if attempt > 10 && w.Code != http.StatusTooManyRequests {
					t.Fatalf("attempt %d bypassed rate limit: HTTP %d, %s", attempt, w.Code, w.Body.String())
				}
			}
			var count int64
			if err := db.Model(&database.AdminLoginRateLimit{}).Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("spoofed headers created extra buckets: %d, %v", count, err)
			}
		})
	}
}
