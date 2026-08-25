package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"joiask-backend/internal/database"
	"joiask-backend/internal/secret"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeAvatarStorage struct{}

func (fakeAvatarStorage) Save(context.Context, int64, string) (string, error) {
	return "/upload-img/test-avatar.jpg", nil
}

func (fakeAvatarStorage) Delete(string) error { return nil }

type testAPIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func performJSONRequest(router http.Handler, method, path, body string, cookies []*http.Cookie) (*httptest.ResponseRecorder, testAPIResponse) {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	for _, item := range cookies {
		request.AddCookie(item)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var response testAPIResponse
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	return recorder, response
}

func TestAccountVerificationRegistrationAndDisabledLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var err error
	database.DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "account.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.AutoMigrate(&database.User{}, &database.BilibiliVerificationAccount{}, &database.BilibiliVerificationRequest{}); err != nil {
		t.Fatal(err)
	}
	viper.Set("security.bilibili_cookie_key", "test-key-that-is-at-least-thirty-two-bytes")
	t.Cleanup(func() { viper.Set("security.bilibili_cookie_key", "") })
	encrypted, err := secret.EncryptCookie("SESSDATA=test")
	if err != nil {
		t.Fatal(err)
	}
	database.DB.Create(&database.BilibiliVerificationAccount{BilibiliUID: 999, EncryptedCookie: encrypted})

	controller := &AccountController{AvatarStorage: fakeAvatarStorage{}}
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-session-secret-that-is-long-enough"))))
	router.POST("/verification", controller.StartVerification)
	router.POST("/register", controller.Register)
	router.POST("/login", controller.Login)

	startRecorder, startResponse := performJSONRequest(router, http.MethodPost, "/verification", `{"bilibili_uid":"123456"}`, nil)
	if startResponse.Code != 200 {
		t.Fatalf("verification start failed: %+v", startResponse)
	}
	var verificationRequest database.BilibiliVerificationRequest
	if err := database.DB.Where("bilibili_uid = ?", 123456).First(&verificationRequest).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	confirmationUntil := now.Add(10 * time.Minute)
	database.DB.Model(&verificationRequest).Updates(map[string]any{
		"status":             database.VerificationVerified,
		"bilibili_name":      "Test User",
		"bilibili_avatar":    "https://example.test/avatar.jpg",
		"verified_at":        now,
		"confirmation_until": confirmationUntil,
	})

	_, registerResponse := performJSONRequest(router, http.MethodPost, "/register", `{"username":"test-user","password":"strong-password"}`, startRecorder.Result().Cookies())
	if registerResponse.Code != 200 {
		t.Fatalf("registration failed: %+v", registerResponse)
	}
	var user database.User
	if err := database.DB.Where("bilibili_uid = ?", 123456).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.BilibiliUID != 123456 {
		t.Fatalf("unexpected user primary key: %d", user.BilibiliUID)
	}
	if user.PasswordHash == "strong-password" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("strong-password")) != nil {
		t.Fatal("password was not stored as a valid bcrypt hash")
	}
	database.DB.Model(&user).Update("is_disabled", true)
	_, loginResponse := performJSONRequest(router, http.MethodPost, "/login", `{"username":"test-user","password":"strong-password"}`, nil)
	if loginResponse.Code != 403 {
		t.Fatalf("disabled login should fail, got %+v", loginResponse)
	}
}
