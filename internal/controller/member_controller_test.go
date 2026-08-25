package controller

import (
	"context"
	"joiask-backend/internal/bilibili"
	"joiask-backend/internal/database"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeBilibiliProfileClient struct{}

func (fakeBilibiliProfileClient) Profile(_ context.Context, uid int64) (bilibili.Profile, error) {
	return bilibili.Profile{MID: uid, Name: "Bilibili User", Face: "https://example.test/avatar.jpg"}, nil
}

func newTestMemberController() *MemberController {
	return &MemberController{Client: fakeBilibiliProfileClient{}, AvatarStorage: fakeAvatarStorage{}}
}

func TestMemberPostCreatesUserWithoutVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var err error
	database.DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "member.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.AutoMigrate(&database.User{}, &database.BilibiliVerificationRequest{}); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/member", newTestMemberController().Post)
	_, response := performJSONRequest(router, http.MethodPost, "/member", `{"bilibili_uid":"123456","username":"manual-user","password":"strong-password"}`, nil)
	if response.Code != 200 {
		t.Fatalf("manual member creation failed: %+v", response)
	}

	var user database.User
	if err := database.DB.First(&user, int64(123456)).Error; err != nil {
		t.Fatal(err)
	}
	if user.Username != "manual-user" || user.BilibiliName != "Bilibili User" || user.BilibiliAvatar == "" {
		t.Fatalf("unexpected manually created user: %+v", user)
	}
	if user.VerifiedAt.IsZero() {
		t.Fatal("manual user should have an administrator-confirmed verification time")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("strong-password")) != nil {
		t.Fatal("manual user password was not bcrypt hashed")
	}
	var verificationCount int64
	if err := database.DB.Model(&database.BilibiliVerificationRequest{}).Count(&verificationCount).Error; err != nil {
		t.Fatal(err)
	}
	if verificationCount != 0 {
		t.Fatalf("manual creation unexpectedly created %d verification requests", verificationCount)
	}
}

func TestMemberPostRejectsDuplicateUIDAndUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var err error
	database.DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "member-duplicates.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.AutoMigrate(&database.User{}); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/member", newTestMemberController().Post)
	_, first := performJSONRequest(router, http.MethodPost, "/member", `{"bilibili_uid":"123456","username":"manual-user","password":"strong-password"}`, nil)
	if first.Code != 200 {
		t.Fatalf("initial member creation failed: %+v", first)
	}
	_, duplicateUID := performJSONRequest(router, http.MethodPost, "/member", `{"bilibili_uid":"123456","username":"another-user","password":"strong-password"}`, nil)
	if duplicateUID.Code != 409 {
		t.Fatalf("duplicate UID should be rejected: %+v", duplicateUID)
	}
	_, duplicateUsername := performJSONRequest(router, http.MethodPost, "/member", `{"bilibili_uid":"654321","username":"manual-user","password":"strong-password"}`, nil)
	if duplicateUsername.Code != 409 {
		t.Fatalf("duplicate username should be rejected: %+v", duplicateUsername)
	}
}
