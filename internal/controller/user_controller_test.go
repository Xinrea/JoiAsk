package controller

import (
	"encoding/json"
	"errors"
	"joiask-backend/internal/adminpassword"
	"joiask-backend/internal/database"
	"joiask-backend/pkg/util"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupAdminTest(t *testing.T) *gin.Engine {
	t.Helper()
	setupRecordIDTestDB(t)
	if err := database.DB.AutoMigrate(&database.AdminLoginRateLimit{}); err != nil {
		t.Fatal(err)
	}
	controller := &UserController{}
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-session-secret-that-is-long-enough"))))
	r.POST("/login", controller.Login)
	r.GET("/session", func(c *gin.Context) {
		admin, ok, err := CurrentAdmin(c)
		if err != nil {
			Fail(c, 500, "内部错误")
			return
		}
		if !ok {
			Fail(c, 401, "未登录")
			return
		}
		Success(c, admin.ID)
	})
	r.POST("/user", func(c *gin.Context) {
		c.Set("user", database.Admin{BaseModel: database.BaseModel{ID: 1}, Username: "operator", Role: database.AdminRoleSuperAdmin})
		controller.Post(c)
	})
	r.PUT("/user/:id", func(c *gin.Context) {
		c.Set("user", database.Admin{BaseModel: database.BaseModel{ID: 1}, Username: "operator", Role: database.AdminRoleSuperAdmin})
		controller.Put(c)
	})
	return r
}

func adminLoginBody(password string) string {
	body, _ := json.Marshal(LoginRequest{Username: "admin", Password: password})
	return string(body)
}

func createTestAdmin(t *testing.T, stored string) database.Admin {
	t.Helper()
	admin := database.Admin{Username: "admin", Password: stored}
	if err := database.DB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	return admin
}

func TestAdminLegacyLoginMigratesWithoutHashReplay(t *testing.T) {
	for _, tc := range []struct {
		name, password string
	}{
		{"ordinary", "old-admin-password"},
		{"long ASCII", strings.Repeat("a", 73)},
		{"long Unicode", strings.Repeat("密", 25)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testAdminLegacyLoginMigratesWithoutHashReplay(t, tc.password)
		})
	}
}

func testAdminLegacyLoginMigratesWithoutHashReplay(t *testing.T, password string) {
	t.Helper()
	r := setupAdminTest(t)
	legacy := util.Md5v(password)
	admin := createTestAdmin(t, legacy)
	for _, input := range []string{"incorrect", legacy} {
		recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody(input), nil)
		if response.Code != 401 || len(recorder.Result().Cookies()) != 0 {
			t.Fatalf("invalid login accepted: %+v", response)
		}
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.Password != legacy {
		t.Fatalf("failed login changed credential: %v", err)
	}
	recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody(password), nil)
	if response.Code != 200 {
		t.Fatalf("legacy raw-password login failed: %+v", response)
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if admin.SessionVersion != 0 {
		t.Fatal("hash migration unexpectedly revoked sessions")
	}
	hash := admin.Password
	if matches, migrate := adminpassword.Verify(hash, password); !matches || migrate {
		t.Fatal("credential not upgraded to a matching non-legacy hash")
	}
	if len(password) > 72 {
		if !strings.HasPrefix(hash, "$bcrypt-sha256$v1$") {
			t.Fatal("long legacy password did not migrate to tagged SHA-256/bcrypt")
		}
	} else if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatalf("ordinary password did not migrate to direct bcrypt: %v", err)
	}
	_, session := performJSONRequest(r, http.MethodGet, "/session", "", recorder.Result().Cookies())
	if session.Code != 200 {
		t.Fatal("successful migration did not establish a session")
	}
	for _, input := range []string{legacy, hash} {
		recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody(input), nil)
		if response.Code != 401 || len(recorder.Result().Cookies()) != 0 {
			t.Fatalf("stored hash replay accepted after migration: %+v", response)
		}
	}
	_, response = performJSONRequest(r, http.MethodPost, "/login", adminLoginBody(password), nil)
	if response.Code != 200 {
		t.Fatalf("subsequent bcrypt login failed: %+v", response)
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.Password != hash {
		t.Fatal("bcrypt login unexpectedly rewrote credential")
	}
}

func TestAdminMigrationFailureDoesNotLogin(t *testing.T) {
	r := setupAdminTest(t)
	admin := createTestAdmin(t, util.Md5v("old-password"))
	migrationAttempted := false
	if err := database.DB.Callback().Update().Before("gorm:update").Register("test:migration_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "admins" {
			migrationAttempted = true
			tx.AddError(errors.New("migration write failed"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("old-password"), nil)
	if !migrationAttempted || response.Code != 500 || len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("migration failure granted session: %+v", response)
	}
	var stored database.Admin
	if err := database.DB.First(&stored, admin.ID).Error; err != nil || stored.Password != admin.Password {
		t.Fatal("migration failure changed password")
	}
}

func TestAdminMigrationDoesNotOverwriteConcurrentReset(t *testing.T) {
	r := setupAdminTest(t)
	admin := createTestAdmin(t, util.Md5v("old-password"))
	newHash, err := adminpassword.Hash("new-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.Callback().Update().Before("gorm:update").Register("test:concurrent_reset", func(tx *gorm.DB) {
		if tx.Statement.Table == "admins" {
			if err := tx.Exec("UPDATE admins SET password = ?, session_version = session_version + 1 WHERE id = ?", newHash, admin.ID).Error; err != nil {
				tx.AddError(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("old-password"), nil)
	if response.Code != 401 || len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("old password granted session after concurrent reset: %+v", response)
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.Password != newHash || admin.SessionVersion != 1 {
		t.Fatal("migration overwrote concurrent reset")
	}
}

func TestAdminUsernameEditDoesNotRevertConcurrentPasswordChange(t *testing.T) {
	r := setupAdminTest(t)
	admin := createTestAdmin(t, util.Md5v("old-password"))
	newHash, err := adminpassword.Hash("new-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.Callback().Update().Before("gorm:update").Register("test:concurrent_password_change", func(tx *gorm.DB) {
		if tx.Statement.Table == "admins" {
			if err := tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE admins SET password = ?, session_version = session_version + 1 WHERE id = ?", newHash, admin.ID).Error; err != nil {
				tx.AddError(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, response := performJSONRequest(r, http.MethodPut, "/user/"+jsonNumber(admin.ID), `{"username":"renamed","password":""}`, nil)
	if response.Code != 200 {
		t.Fatalf("username edit failed: %+v", response)
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.Username != "renamed" || admin.Password != newHash || admin.SessionVersion != 1 {
		t.Fatal("username-only edit reverted concurrent password change")
	}
}

func TestAdminLoginHandlerEnforcesRateBeforeCredentialLookup(t *testing.T) {
	r := setupAdminTest(t)
	queries := 0
	if err := database.DB.Callback().Query().Before("gorm:query").Register("test:login_lookup_count", func(tx *gorm.DB) {
		if tx.Statement.Table == "admins" {
			queries++
		}
	}); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 30; attempt++ {
		recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("wrong-password"), nil)
		if attempt <= 10 {
			if response.Code != 401 {
				t.Fatalf("attempt %d: %+v", attempt, response)
			}
		} else if recorder.Code != http.StatusTooManyRequests || response.Code != 429 {
			t.Fatalf("attempt %d not rate limited: HTTP %d, %+v", attempt, recorder.Code, response)
		}
		if len(recorder.Result().Cookies()) != 0 {
			t.Fatal("failed or rate-limited login issued a session")
		}
	}
	if queries != 10 {
		t.Fatalf("rate-limited attempts reached credential lookup: %d queries", queries)
	}
}

func TestAdminCredentialWrites(t *testing.T) {
	r := setupAdminTest(t)
	_, response := performJSONRequest(r, http.MethodPost, "/user", `{"username":"admin","password":"new-password"}`, nil)
	if response.Code != 200 || strings.Contains(string(response.Data), "password") {
		t.Fatalf("create failed or leaked password: %+v", response)
	}
	var admin database.Admin
	if err := database.DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("new-password")) != nil {
		t.Fatal("create did not hash password")
	}
	hash := admin.Password
	path := "/user/" + jsonNumber(admin.ID)
	for _, body := range []string{`{"username":"admin"}`, `{"password":""}`} {
		_, response = performJSONRequest(r, http.MethodPut, path, body, nil)
		if response.Code != 200 {
			t.Fatalf("empty password edit failed: %+v", response)
		}
		if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.Password != hash || admin.SessionVersion != 0 {
			t.Fatal("empty password changed credential")
		}
	}
	_, response = performJSONRequest(r, http.MethodPut, path, `{"password":"changed-password"}`, nil)
	if response.Code != 200 {
		t.Fatalf("password edit failed: %+v", response)
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.SessionVersion != 1 ||
		bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("changed-password")) != nil {
		t.Fatal("edit did not hash new password and advance session version")
	}
	_, oldLogin := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("new-password"), nil)
	_, newLogin := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("changed-password"), nil)
	if oldLogin.Code != 401 || newLogin.Code != 200 {
		t.Fatalf("password change login behavior: old=%+v new=%+v", oldLogin, newLogin)
	}
}

func TestAdminConcurrentResetDoesNotUpgradeLoginSession(t *testing.T) {
	r := setupAdminTest(t)
	oldHash, err := adminpassword.Hash("old-password")
	if err != nil {
		t.Fatal(err)
	}
	newHash, err := adminpassword.Hash("new-password")
	if err != nil {
		t.Fatal(err)
	}
	admin := createTestAdmin(t, oldHash)
	reset := false
	if err := database.DB.Callback().Query().After("gorm:query").Register("test:reset_after_login_lookup", func(tx *gorm.DB) {
		if tx.Statement.Table == "admins" && !reset {
			reset = true
			// Simulate a reset after Login has read the old password/version but
			// before it verifies that snapshot and issues its cookie.
			if err := tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE admins SET password = ?, session_version = session_version + 1 WHERE id = ?", newHash, admin.ID).Error; err != nil {
				tx.AddError(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("old-password"), nil)
	if !reset || response.Code != 200 {
		t.Fatalf("expected login using the pre-reset snapshot: %+v", response)
	}
	_, session := performJSONRequest(r, http.MethodGet, "/session", "", recorder.Result().Cookies())
	if session.Code != 401 {
		t.Fatalf("pre-reset credentials received a post-reset session: %+v", session)
	}
	recorder, response = performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("new-password"), nil)
	if response.Code != 200 {
		t.Fatalf("new password login failed: %+v", response)
	}
	_, session = performJSONRequest(r, http.MethodGet, "/session", "", recorder.Result().Cookies())
	if session.Code != 200 {
		t.Fatalf("new password received an invalid session: %+v", session)
	}
}

func TestAdminPasswordResetFailurePreservesCredentialAndSession(t *testing.T) {
	r := setupAdminTest(t)
	hash, err := adminpassword.Hash("old-password")
	if err != nil {
		t.Fatal(err)
	}
	admin := createTestAdmin(t, hash)
	login, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("old-password"), nil)
	if response.Code != 200 {
		t.Fatalf("login failed: %+v", response)
	}
	if err := database.DB.Callback().Update().After("gorm:update").Before("gorm:commit_or_rollback_transaction").Register("test:reset_rollback", func(tx *gorm.DB) {
		if tx.Statement.Table == "admins" {
			tx.AddError(errors.New("reset write failed"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, response = performJSONRequest(r, http.MethodPut, "/user/"+jsonNumber(admin.ID), `{"password":"new-password"}`, nil)
	if response.Code != 500 {
		t.Fatalf("failed reset reported success: %+v", response)
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.Password != hash || admin.SessionVersion != 0 {
		t.Fatal("failed reset changed password or session version")
	}
	_, session := performJSONRequest(r, http.MethodGet, "/session", "", login.Result().Cookies())
	if session.Code != 200 {
		t.Fatalf("failed reset revoked an existing session: %+v", session)
	}
}

func TestAdminPasswordResetPreservesConcurrentVersionIncrement(t *testing.T) {
	r := setupAdminTest(t)
	admin := createTestAdmin(t, util.Md5v("old-password"))
	if err := database.DB.Callback().Update().Before("gorm:update").Register("test:concurrent_version_increment", func(tx *gorm.DB) {
		if tx.Statement.Table == "admins" {
			if err := tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE admins SET session_version = session_version + 1 WHERE id = ?", admin.ID).Error; err != nil {
				tx.AddError(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, response := performJSONRequest(r, http.MethodPut, "/user/"+jsonNumber(admin.ID), `{"password":"new-password"}`, nil)
	if response.Code != 200 {
		t.Fatalf("password reset failed: %+v", response)
	}
	if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.SessionVersion != 2 {
		t.Fatal("password reset lost a concurrent session version increment")
	}
	if matches, _ := adminpassword.Verify(admin.Password, "new-password"); !matches {
		t.Fatal("password reset did not persist the new credential")
	}
}

func jsonNumber(id uint) string {
	number, _ := json.Marshal(id)
	return string(number)
}

func TestAdminLongPasswordCreateAndChange(t *testing.T) {
	r := setupAdminTest(t)
	first := strings.Repeat("密", 25)
	second := strings.Repeat("a", 72) + "different-tail"
	_, response := performJSONRequest(r, http.MethodPost, "/user", adminLoginBody(first), nil)
	if response.Code != 200 {
		t.Fatalf("create with long password failed: %+v", response)
	}
	var admin database.Admin
	if err := database.DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	assertStored := func(password string, tagged bool) {
		t.Helper()
		if err := database.DB.First(&admin, admin.ID).Error; err != nil {
			t.Fatal(err)
		}
		matches, migrate := adminpassword.Verify(admin.Password, password)
		if !matches || migrate || strings.HasPrefix(admin.Password, "$bcrypt-sha256$v1$") != tagged {
			t.Fatal("password not stored in the expected bcrypt format")
		}
	}
	assertLogin := func(password string, want int) {
		t.Helper()
		recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody(password), nil)
		if response.Code != want || (want != 200 && len(recorder.Result().Cookies()) != 0) {
			t.Fatalf("login: got %+v, want %d", response, want)
		}
	}
	assertStored(first, true)
	assertLogin(first, 200)
	assertLogin(first[:72], 401)
	path := "/user/" + jsonNumber(admin.ID)
	for _, password := range []string{second, "ordinary-password", first} {
		_, response = performJSONRequest(r, http.MethodPut, path, adminLoginBody(password), nil)
		if response.Code != 200 {
			t.Fatalf("password change failed: %+v", response)
		}
		assertStored(password, len(password) > 72)
		assertLogin(password, 200)
		assertLogin(admin.Password, 401)
	}
	// Changing to a longer password must not preserve the first 72 bytes as an
	// alternative credential, or accept a different suffix.
	assertLogin(first+"wrong-tail", 401)
}

func TestAdminLoginRejectsUntaggedBcryptTruncation(t *testing.T) {
	for _, password := range []string{strings.Repeat("a", 72), strings.Repeat("密", 24)} {
		t.Run(password, func(t *testing.T) {
			r := setupAdminTest(t)
			hash, err := adminpassword.Hash(password)
			if err != nil {
				t.Fatal(err)
			}
			admin := createTestAdmin(t, hash)
			recorder, response := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody(password+"suffix"), nil)
			if response.Code != 401 || len(recorder.Result().Cookies()) != 0 {
				t.Fatalf("overlong input matched untagged bcrypt: %+v", response)
			}
			if err := database.DB.First(&admin, admin.ID).Error; err != nil || admin.Password != hash {
				t.Fatal("rejected overlong input changed stored credential")
			}
			_, response = performJSONRequest(r, http.MethodPost, "/login", adminLoginBody(password), nil)
			if response.Code != 200 {
				t.Fatalf("original 72-byte password no longer works: %+v", response)
			}
		})
	}
}

func TestAdminInvalidCredentials(t *testing.T) {
	r := setupAdminTest(t)
	hash, err := adminpassword.Hash("password")
	if err != nil {
		t.Fatal(err)
	}
	createTestAdmin(t, hash)
	_, wrong := performJSONRequest(r, http.MethodPost, "/login", adminLoginBody("wrong"), nil)
	_, unknown := performJSONRequest(r, http.MethodPost, "/login", `{"username":"unknown","password":"wrong"}`, nil)
	if wrong.Code != 401 || unknown.Code != wrong.Code || unknown.Message != wrong.Message {
		t.Fatal("unknown account and wrong password should have the same error")
	}
	for _, body := range []string{
		`{`, `{}`, `{"username":"admin","password":""}`,
	} {
		recorder, response := performJSONRequest(r, http.MethodPost, "/login", body, nil)
		if response.Code != 400 || len(recorder.Result().Cookies()) != 0 {
			t.Fatalf("invalid request accepted: %+v", response)
		}
	}
	_, response := performJSONRequest(r, http.MethodPost, "/user", adminLoginBody(""), nil)
	if response.Code != 400 {
		t.Fatalf("empty create password accepted: %+v", response)
	}
}
