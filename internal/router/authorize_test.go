package router

import (
	"encoding/json"
	"fmt"
	"joiask-backend/internal/adminpassword"
	"joiask-backend/internal/controller"
	"joiask-backend/internal/database"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const authorizePassword = "original-admin-password"

type authorizeFixture struct {
	router    *gin.Engine
	store     sessions.Store
	db        *gorm.DB
	admin     database.Admin
	editor    database.Admin
	published database.Question
	draft     database.Question
}

func setupAuthorizeTest(t *testing.T) *authorizeFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "authorize.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
		sqlDB.Close()
	})
	if err := db.AutoMigrate(&database.Admin{}, &database.AdminLoginRateLimit{}, &database.Tag{}, &database.Question{}); err != nil {
		t.Fatal(err)
	}
	hash, err := adminpassword.Hash(authorizePassword)
	if err != nil {
		t.Fatal(err)
	}
	f := &authorizeFixture{
		db:     db,
		admin:  database.Admin{Username: "admin", Password: hash},
		editor: database.Admin{Username: "editor", Password: hash},
	}
	for _, admin := range []*database.Admin{&f.admin, &f.editor} {
		if err := db.Create(admin).Error; err != nil {
			t.Fatal(err)
		}
	}
	tag := database.Tag{TagName: "authorization"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatal(err)
	}
	f.published = database.Question{TagID: int(tag.ID), Content: "public question", IsPublish: true}
	f.draft = database.Question{TagID: int(tag.ID), Content: "unpublished question"}
	for _, question := range []*database.Question{&f.published, &f.draft} {
		if err := db.Create(question).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.store, err = NewSessionStore(randomSessionKey(t), false)
	if err != nil {
		t.Fatal(err)
	}
	f.router = gin.New()
	f.router.Use(sessions.Sessions("session", f.store))
	users := &controller.UserController{}
	// Do not construct the background broadcast workers for a list-only test.
	questions := &controller.QuestionController{}
	f.router.POST("/api/login", users.Login)
	f.router.GET("/api/info", authMiddleware, users.Info)
	f.router.PUT("/api/user/:id", authMiddleware, users.Put)
	f.router.DELETE("/api/user/:id", authMiddleware, users.Delete)
	f.router.GET("/api/question", optionalAuthMiddleware, questions.Get)
	f.router.GET("/api/logout", authMiddleware, users.Logout)
	return f
}

func (f *authorizeFixture) request(t *testing.T, method, path, body string, cookie *http.Cookie, wantCode int) (*httptest.ResponseRecorder, json.RawMessage) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.1:1234"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, req)
	var response struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("%s %s: invalid JSON: %v: %s", method, path, err, recorder.Body.String())
	}
	if recorder.Code != http.StatusOK || response.Code != wantCode {
		t.Fatalf("%s %s: HTTP %d, body %s; want HTTP 200, business code %d", method, path, recorder.Code, recorder.Body.String(), wantCode)
	}
	return recorder, response.Data
}

func authorizeResponseCookie(t *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}
	t.Fatal("response did not save a session cookie")
	return nil
}

func (f *authorizeFixture) sessionValues(t *testing.T, cookie *http.Cookie) map[interface{}]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	session, err := f.store.Get(req, "session")
	if err != nil {
		t.Fatal(err)
	}
	return session.Values
}

// Construct real signed cookies to exercise legacy and malformed session values
// without replacing either middleware or injecting trusted Gin context values.
func (f *authorizeFixture) signedCookie(t *testing.T, values map[string]interface{}) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	session, err := f.store.Get(req, "session")
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range values {
		session.Values[key] = value
	}
	recorder := httptest.NewRecorder()
	if err := f.store.Save(req, recorder, session); err != nil {
		t.Fatal(err)
	}
	return authorizeResponseCookie(t, recorder)
}

func (f *authorizeFixture) login(t *testing.T, admin database.Admin, password string, existing *http.Cookie) *http.Cookie {
	t.Helper()
	body, err := json.Marshal(controller.LoginRequest{Username: admin.Username, Password: password})
	if err != nil {
		t.Fatal(err)
	}
	recorder, _ := f.request(t, http.MethodPost, "/api/login", string(body), existing, 200)
	cookie := authorizeResponseCookie(t, recorder)
	values := f.sessionValues(t, cookie)
	if values["user"] != admin.ID || values["authed"] != true || values["admin_session_version"] != admin.SessionVersion {
		t.Fatalf("login did not save strictly typed admin fields: %#v", values)
	}
	return cookie
}

func (f *authorizeFixture) assertAdminInfo(t *testing.T, cookie *http.Cookie, admin database.Admin) {
	t.Helper()
	_, data := f.request(t, http.MethodGet, "/api/info", "", cookie, 200)
	var info struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatal(err)
	}
	if info.ID != admin.ID || info.Username != admin.Username {
		t.Fatalf("info = %+v, want admin %d/%s", info, admin.ID, admin.Username)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"password", "session_version", "SessionVersion", "admin_session_version"} {
		if _, exists := fields[private]; exists {
			t.Fatalf("info exposed private field %s: %s", private, data)
		}
	}
}

func (f *authorizeFixture) assertQuestionVisibility(t *testing.T, cookie *http.Cookie, isAdmin bool) {
	t.Helper()
	for _, query := range []string{"", "?publish=false"} {
		_, data := f.request(t, http.MethodGet, "/api/question"+query, "", cookie, 200)
		var list struct {
			Questions []database.Question `json:"questions"`
			Total     int                 `json:"total"`
		}
		if err := json.Unmarshal(data, &list); err != nil {
			t.Fatal(err)
		}
		want := map[uint]bool{f.published.ID: true}
		if isAdmin {
			want[f.draft.ID] = true
			if query != "" {
				delete(want, f.published.ID)
			}
		}
		if list.Total != len(want) || len(list.Questions) != len(want) {
			t.Fatalf("admin=%t query=%q: wrong visibility: %s", isAdmin, query, data)
		}
		for _, question := range list.Questions {
			if !want[question.ID] {
				t.Fatalf("admin=%t query=%q: unexpected question %d: %s", isAdmin, query, question.ID, data)
			}
			delete(want, question.ID)
			if question.Tag.TagName != "authorization" {
				t.Fatalf("question tag was not preloaded: %s", data)
			}
		}
	}
}

func TestAdminAuthorizationPasswordResetRevokesCopiedCookie(t *testing.T) {
	f := setupAuthorizeTest(t)
	current := f.login(t, f.admin, authorizePassword, nil)
	copied := *current
	f.assertAdminInfo(t, current, f.admin)
	f.assertQuestionVisibility(t, current, true)
	f.assertAdminInfo(t, &copied, f.admin)
	f.assertQuestionVisibility(t, &copied, true)

	// Reset through the protected endpoint as the default administrator, not
	// through a handler with a fabricated privileged context.
	f.request(t, http.MethodPut, fmt.Sprintf("/api/user/%d", f.admin.ID), `{"password":"replacement-password"}`, current, 200)
	var updated database.Admin
	if err := f.db.First(&updated, f.admin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.SessionVersion != f.admin.SessionVersion+1 {
		t.Fatalf("password reset version = %d, want %d", updated.SessionVersion, f.admin.SessionVersion+1)
	}
	for name, cookie := range map[string]*http.Cookie{"current": current, "copied": &copied} {
		t.Run(name, func(t *testing.T) {
			f.request(t, http.MethodGet, "/api/info", "", cookie, 408)
			// Replay the original cookie even if /info returned a clearing cookie.
			f.assertQuestionVisibility(t, cookie, false)
			f.request(t, http.MethodPut, fmt.Sprintf("/api/user/%d", f.editor.ID), `{"password":"unauthorized"}`, cookie, 408)
			f.request(t, http.MethodDelete, fmt.Sprintf("/api/user/%d", f.editor.ID), "", cookie, 408)
		})
	}
	body, err := json.Marshal(controller.LoginRequest{Username: "admin", Password: authorizePassword})
	if err != nil {
		t.Fatal(err)
	}
	recorder, _ := f.request(t, http.MethodPost, "/api/login", string(body), nil, 401)
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatal("old password issued a session after reset")
	}
	fresh := f.login(t, updated, "replacement-password", nil)
	f.assertAdminInfo(t, fresh, updated)
	f.assertQuestionVisibility(t, fresh, true)
}

func TestAdminAuthorizationDeletionRevokesListPrivileges(t *testing.T) {
	f := setupAuthorizeTest(t)
	adminCookie := f.login(t, f.admin, authorizePassword, nil)
	editorCookie := f.login(t, f.editor, authorizePassword, nil)
	f.assertAdminInfo(t, editorCookie, f.editor)
	f.assertQuestionVisibility(t, editorCookie, true)
	f.request(t, http.MethodDelete, fmt.Sprintf("/api/user/%d", f.editor.ID), "", adminCookie, 200)
	f.request(t, http.MethodGet, "/api/info", "", editorCookie, 408)
	f.assertQuestionVisibility(t, editorCookie, false)
	f.assertAdminInfo(t, adminCookie, f.admin)
	f.assertQuestionVisibility(t, nil, false)
}

func TestAdminAuthorizationStrictSessionFields(t *testing.T) {
	f := setupAuthorizeTest(t)
	for _, version := range []int{0, 3} {
		t.Run(fmt.Sprintf("database version %d", version), func(t *testing.T) {
			if err := f.db.Model(&f.admin).Update("session_version", version).Error; err != nil {
				t.Fatal(err)
			}
			f.admin.SessionVersion = version
			for _, tc := range []struct {
				name  string
				field string
				value interface{}
			}{
				{"legacy missing version", "admin_session_version", nil},
				{"string version", "admin_session_version", fmt.Sprint(version)},
				{"uint version", "admin_session_version", uint(version)},
				{"int64 version", "admin_session_version", int64(version)},
				{"mismatched version", "admin_session_version", version + 1},
				{"negative version", "admin_session_version", -1},
				{"missing user", "user", nil},
				{"string user", "user", fmt.Sprint(f.admin.ID)},
				{"int user", "user", int(f.admin.ID)},
				{"zero user", "user", uint(0)},
				{"nonexistent user", "user", f.editor.ID + 100},
				{"missing authed", "authed", nil},
				{"false authed", "authed", false},
				{"string authed", "authed", "true"},
				{"int authed", "authed", 1},
			} {
				t.Run(tc.name, func(t *testing.T) {
					values := map[string]interface{}{"user": f.admin.ID, "authed": true, "admin_session_version": version}
					if tc.value == nil {
						delete(values, tc.field)
					} else {
						values[tc.field] = tc.value
					}
					cookie := f.signedCookie(t, values)
					f.request(t, http.MethodGet, "/api/info", "", cookie, 408)
					f.assertQuestionVisibility(t, cookie, false)
				})
			}
			valid := f.signedCookie(t, map[string]interface{}{"user": f.admin.ID, "authed": true, "admin_session_version": version})
			f.assertAdminInfo(t, valid, f.admin)
			f.assertQuestionVisibility(t, valid, true)
			f.assertQuestionVisibility(t, nil, false)
		})
	}
}

func TestAdminAuthorizationEmptyPasswordPreservesSession(t *testing.T) {
	f := setupAuthorizeTest(t)
	adminCookie := f.login(t, f.admin, authorizePassword, nil)
	editorCookie := f.login(t, f.editor, authorizePassword, nil)
	originalHash := f.editor.Password
	for _, body := range []string{`{"username":"renamed-editor"}`, `{"password":""}`} {
		f.request(t, http.MethodPut, fmt.Sprintf("/api/user/%d", f.editor.ID), body, adminCookie, 200)
		var updated database.Admin
		if err := f.db.First(&updated, f.editor.ID).Error; err != nil {
			t.Fatal(err)
		}
		if updated.Username != "renamed-editor" || updated.Password != originalHash || updated.SessionVersion != f.editor.SessionVersion {
			t.Fatalf("username/empty-password edit changed credential or version: %+v", updated)
		}
		f.assertAdminInfo(t, editorCookie, updated)
		f.assertQuestionVisibility(t, editorCookie, true)
	}
}

func TestAdminAuthorizationLogoutPreservesMemberSession(t *testing.T) {
	f := setupAuthorizeTest(t)
	memberValues := map[string]interface{}{
		"member_bilibili_uid":         int64(123456),
		"member_session_version":      7,
		"bilibili_verification_token": "unrelated-verification-token",
	}
	memberCookie := f.signedCookie(t, memberValues)
	adminCookie := f.login(t, f.admin, authorizePassword, memberCookie)
	recorder, _ := f.request(t, http.MethodGet, "/api/logout", "", adminCookie, 200)
	loggedOut := authorizeResponseCookie(t, recorder)
	values := f.sessionValues(t, loggedOut)
	for _, key := range []string{"user", "authed", "admin_session_version"} {
		if _, exists := values[key]; exists {
			t.Fatalf("logout retained admin field %q: %#v", key, values)
		}
	}
	for key, want := range memberValues {
		if !reflect.DeepEqual(values[key], want) {
			t.Fatalf("logout changed unrelated field %q: got %#v, want %#v", key, values[key], want)
		}
	}
	f.request(t, http.MethodGet, "/api/info", "", loggedOut, 408)
	f.assertQuestionVisibility(t, loggedOut, false)
}

func TestAdminAuthorizationDatabaseFailureFailsClosed(t *testing.T) {
	f := setupAuthorizeTest(t)
	cookie := f.login(t, f.admin, authorizePassword, nil)
	// Keep the question/tag tables usable so an optional-auth lookup failure
	// cannot be mistaken for a successful guest list or an admin list query error.
	if err := f.db.Migrator().DropTable(&database.Admin{}); err != nil {
		t.Fatal(err)
	}
	f.request(t, http.MethodGet, "/api/info", "", cookie, 500)
	f.request(t, http.MethodGet, "/api/question", "", cookie, 500)
	f.request(t, http.MethodGet, "/api/question?publish=false", "", cookie, 500)
	f.assertQuestionVisibility(t, nil, false)
}
