package router

import (
	"encoding/json"
	"fmt"
	"joiask-backend/internal/database"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuperAdministratorCanManageAccountsAfterRename(t *testing.T) {
	f := setupAuthorizeTest(t)
	cookie := f.login(t, f.admin, authorizePassword, nil)
	path := fmt.Sprintf("/api/user/%d", f.admin.ID)
	f.request(t, http.MethodPut, path, `{"username":"renamed","password":""}`, cookie, 200)
	f.admin.Username = "renamed"
	f.assertAdminInfo(t, cookie, f.admin)

	// Both the original session and a fresh login retain the explicit role.
	for _, session := range []*http.Cookie{cookie, f.login(t, f.admin, authorizePassword, nil)} {
		f.request(t, http.MethodGet, "/api/user", "", session, 200)
		_, data := f.request(t, http.MethodPost, "/api/user", `{"username":"colleague","password":"colleague-password"}`, session, 200)
		var created database.Admin
		if err := json.Unmarshal(data, &created); err != nil || created.ID == 0 || created.Role != database.AdminRoleAdmin {
			t.Fatalf("invalid created administrator: %s, %v", data, err)
		}
		target := fmt.Sprintf("/api/user/%d", created.ID)
		f.request(t, http.MethodPut, target, `{"username":"updated","password":"reset-password"}`, session, 200)
		f.request(t, http.MethodDelete, target, "", session, 200)
	}

	// "admin" is just a login name, not a protected account or role grant.
	editorPath := fmt.Sprintf("/api/user/%d", f.editor.ID)
	f.request(t, http.MethodPut, editorPath, `{"username":"admin"}`, cookie, 200)
	f.request(t, http.MethodDelete, editorPath, "", cookie, 200)
	f.request(t, http.MethodDelete, path, "", cookie, 403)
	var count int64
	if err := f.db.Model(&database.Admin{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected the current administrator to remain: count=%d err=%v", count, err)
	}
}

func TestAdminCannotDeleteSelfRegardlessOfUsername(t *testing.T) {
	for _, name := range []string{"admin", "operator"} {
		t.Run(name, func(t *testing.T) {
			f := setupAuthorizeTest(t)
			cookie := f.login(t, f.admin, authorizePassword, nil)
			path := fmt.Sprintf("/api/user/%d", f.admin.ID)
			f.request(t, http.MethodPut, path, fmt.Sprintf(`{"username":%q}`, name), cookie, 200)
			f.request(t, http.MethodDelete, path, "", cookie, 403)
			var remaining database.Admin
			if err := f.db.First(&remaining, f.admin.ID).Error; err != nil || remaining.Username != name {
				t.Fatalf("current account was removed: %+v, %v", remaining, err)
			}
		})
	}
}

func TestAdminManagementRequiresValidSession(t *testing.T) {
	f := setupAuthorizeTest(t)
	deletedCookie := f.login(t, f.editor, authorizePassword, nil)
	if err := f.db.Delete(&f.editor).Error; err != nil {
		t.Fatal(err)
	}
	for _, cookie := range []*http.Cookie{nil, deletedCookie} {
		for _, request := range []struct {
			method, path, body string
		}{
			{http.MethodGet, "/api/user", ""},
			{http.MethodPost, "/api/user", `{"username":"intruder","password":"password"}`},
			{http.MethodPut, fmt.Sprintf("/api/user/%d", f.admin.ID), `{"username":"intruder"}`},
			{http.MethodDelete, fmt.Sprintf("/api/user/%d", f.admin.ID), ""},
		} {
			f.request(t, request.method, request.path, request.body, cookie, 408)
		}
	}
}

func TestOrdinaryAdministratorCanOnlyEditOwnProfile(t *testing.T) {
	f := setupAuthorizeTest(t)
	superCookie := f.login(t, f.admin, authorizePassword, nil)
	// Free the legacy login name, then let an ordinary admin take it.
	f.request(t, http.MethodPut, fmt.Sprintf("/api/user/%d", f.admin.ID), `{"username":"owner"}`, superCookie, 200)
	cookie := f.login(t, f.editor, authorizePassword, nil)
	self := fmt.Sprintf("/api/user/%d", f.editor.ID)
	f.request(t, http.MethodPut, self, `{"username":"admin"}`, cookie, 200)
	f.editor.Username = "admin"
	f.assertAdminInfo(t, cookie, f.editor)
	f.assertQuestionVisibility(t, cookie, true)
	f.request(t, http.MethodGet, "/api/user", "", cookie, 403)
	f.request(t, http.MethodPost, "/api/user", `{"username":"extra","password":"password","role":"super_admin"}`, cookie, 403)
	f.request(t, http.MethodPut, fmt.Sprintf("/api/user/%d", f.admin.ID), `{"password":"changed"}`, cookie, 403)
	f.request(t, http.MethodDelete, fmt.Sprintf("/api/user/%d", f.admin.ID), "", cookie, 403)
	for _, role := range []string{"super_admin", "admin", ""} {
		f.request(t, http.MethodPut, self, fmt.Sprintf(`{"role":%q}`, role), cookie, 403)
	}
	f.request(t, http.MethodPut, self, `{"password":"own-new-password"}`, cookie, 200)
	f.request(t, http.MethodGet, "/api/info", "", cookie, 408)
	f.editor.SessionVersion++
	cookie = f.login(t, f.editor, "own-new-password", nil)
	f.assertAdminInfo(t, cookie, f.editor)
	f.request(t, http.MethodGet, "/api/user", "", cookie, 403)
	var stored database.Admin
	if err := f.db.First(&stored, f.editor.ID).Error; err != nil || stored.Role != database.AdminRoleAdmin {
		t.Fatalf("ordinary administrator gained a role: %+v, %v", stored, err)
	}
}

func TestRoleChangesRevokeSessionsAndProtectLastSuperAdmin(t *testing.T) {
	f := setupAuthorizeTest(t)
	cookie := f.login(t, f.admin, authorizePassword, nil)
	editorCookie := f.login(t, f.editor, authorizePassword, nil)
	self := fmt.Sprintf("/api/user/%d", f.admin.ID)
	other := fmt.Sprintf("/api/user/%d", f.editor.ID)
	f.request(t, http.MethodPut, self, `{"role":"admin"}`, cookie, 403)
	f.request(t, http.MethodPut, self, `{"role":"owner"}`, cookie, 400)
	f.request(t, http.MethodPost, "/api/user", `{"username":"invalid","password":"password","role":"owner"}`, cookie, 400)
	f.request(t, http.MethodPut, other, `{"role":"super_admin"}`, cookie, 200)
	f.request(t, http.MethodGet, "/api/info", "", editorCookie, 408)
	f.editor.Role = database.AdminRoleSuperAdmin
	f.editor.SessionVersion++
	editorCookie = f.login(t, f.editor, authorizePassword, nil)
	f.request(t, http.MethodGet, "/api/user", "", editorCookie, 200)
	// Demotion is allowed once a second super admin exists, but revokes cookies.
	f.request(t, http.MethodPut, self, `{"role":"admin"}`, cookie, 200)
	f.request(t, http.MethodGet, "/api/info", "", cookie, 408)
	f.admin.Role = database.AdminRoleAdmin
	f.admin.SessionVersion++
	cookie = f.login(t, f.admin, authorizePassword, nil)
	f.request(t, http.MethodGet, "/api/user", "", cookie, 403)
	f.request(t, http.MethodPut, other, `{"role":"admin"}`, editorCookie, 403)
	f.request(t, http.MethodDelete, other, "", editorCookie, 403)
	// Re-submitting an unchanged role doesn't unnecessarily revoke sessions.
	f.request(t, http.MethodPut, other, `{"role":"super_admin"}`, editorCookie, 200)
	f.assertAdminInfo(t, editorCookie, f.editor)
}

func TestConcurrentSelfDemotionsKeepOneSuperAdministrator(t *testing.T) {
	f := setupAuthorizeTest(t)
	if err := f.db.Model(&f.editor).UpdateColumn("role", database.AdminRoleSuperAdmin).Error; err != nil {
		t.Fatal(err)
	}
	f.editor.Role = database.AdminRoleSuperAdmin
	cookies := []*http.Cookie{
		f.login(t, f.admin, authorizePassword, nil),
		f.login(t, f.editor, authorizePassword, nil),
	}
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	for i, admin := range []database.Admin{f.admin, f.editor} {
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/user/%d", admin.ID), strings.NewReader(`{"role":"admin"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookies[i])
		go func() {
			<-start
			w := httptest.NewRecorder()
			f.router.ServeHTTP(w, req)
			results <- w
		}()
	}
	close(start)
	successes := 0
	for range cookies {
		w := <-results
		var response struct{ Code int }
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Code == 200 {
			successes++
		}
	}
	var count int64
	if err := f.db.Model(&database.Admin{}).Where("role = ?", database.AdminRoleSuperAdmin).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if successes != 1 || count != 1 {
		t.Fatalf("concurrent demotions: successes=%d super administrators=%d", successes, count)
	}
}
