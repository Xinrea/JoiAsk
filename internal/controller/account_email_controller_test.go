package controller

import (
	"context"
	"encoding/json"
	"errors"
	"joiask-backend/internal/database"
	"joiask-backend/internal/mailer"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type capturedEmail struct{ to, subject, body string }
type fakeMailSender struct {
	messages []capturedEmail
	err      error
}

func (f *fakeMailSender) Send(_ context.Context, _ mailer.Settings, to, subject, body string) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, capturedEmail{to, subject, body})
	return nil
}
func (f *fakeMailSender) token(t *testing.T) string {
	t.Helper()
	if len(f.messages) == 0 {
		t.Fatal("expected an email")
	}
	for _, line := range strings.Split(f.messages[len(f.messages)-1].body, "\n") {
		if strings.HasPrefix(line, "https://") {
			u, err := url.Parse(line)
			if err != nil {
				t.Fatal(err)
			}
			values, _ := url.ParseQuery(u.Fragment)
			token := values.Get("token")
			if len(token) != 64 {
				t.Fatal("missing credential in link")
			}
			return token
		}
	}
	t.Fatal("missing email link")
	return ""
}

func setupEmailTest(t *testing.T) (*gin.Engine, *fakeMailSender, []*http.Cookie) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var err error
	database.DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "email.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.AutoMigrate(&database.Config{}, &database.User{}, &database.EmailToken{}, &database.EmailRateLimit{}); err != nil {
		t.Fatal(err)
	}
	config := database.Config{SiteName: "测试提问箱", SpamPrompt: "垃圾广告", EmailEnabled: true, SMTPHost: "smtp.example.test", SMTPPort: 587, SMTPSecurity: "starttls", SMTPUsername: "sender", SMTPPassword: "private-password", SMTPFromEmail: "sender@example.test", SMTPFromName: "提问箱", EmailSiteURL: "https://ask.example.test"}
	if err := database.DB.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []database.User{{BilibiliUID: 1, Username: "alice", PasswordHash: string(hash), VerifiedAt: time.Now()}, {BilibiliUID: 2, Username: "bob", PasswordHash: string(hash), VerifiedAt: time.Now()}} {
		if err := database.DB.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	sender := &fakeMailSender{}
	ctl := &AccountController{MailSender: sender}
	configCtl := &ConfigController{}
	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-session-secret-that-is-long-enough"))))
	r.POST("/login", ctl.Login)
	r.GET("/info", ctl.Info)
	r.POST("/bind", ctl.RequestEmailBinding)
	r.POST("/bind/confirm", ctl.ConfirmEmailBinding)
	r.POST("/reset", ctl.RequestPasswordReset)
	r.POST("/reset/confirm", ctl.ResetPassword)
	r.GET("/config", configCtl.Get)
	r.GET("/settings", configCtl.GetSettings)
	r.PUT("/settings", configCtl.PutSettings)
	login, response := performJSONRequest(r, "POST", "/login", `{"username":"alice","password":"old-password"}`, nil)
	if response.Code != 200 {
		t.Fatalf("login failed: %+v", response)
	}
	return r, sender, login.Result().Cookies()
}
func clearEmailLimits(t *testing.T) {
	t.Helper()
	if err := database.DB.Where("1 = 1").Delete(&database.EmailRateLimit{}).Error; err != nil {
		t.Fatal(err)
	}
}
func requireEmailCode(t *testing.T, got testAPIResponse, code int) {
	t.Helper()
	if got.Code != code {
		t.Fatalf("expected %d, got %+v", code, got)
	}
}
func confirmBody(token string) string {
	raw, _ := json.Marshal(gin.H{"token": token})
	return string(raw)
}
func resetBody(token, password string) string {
	raw, _ := json.Marshal(gin.H{"token": token, "password": password})
	return string(raw)
}

func TestEmailBindingAndPasswordReset(t *testing.T) {
	r, sender, cookies := setupEmailTest(t)
	_, response := performJSONRequest(r, "POST", "/bind", `{"email":"Alice@Example.test","password":"old-password"}`, nil)
	requireEmailCode(t, response, 408)
	_, response = performJSONRequest(r, "POST", "/bind", `{"email":"Alice@Example.test","password":"wrong"}`, cookies)
	requireEmailCode(t, response, 403)
	clearEmailLimits(t)
	_, response = performJSONRequest(r, "POST", "/bind", `{"email":" Alice@Example.test ","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 200)
	bindingToken := sender.token(t)
	var user database.User
	database.DB.First(&user, 1)
	if user.Email != nil {
		t.Fatal("email bound before confirmation")
	}
	if sender.messages[0].to != "alice@example.test" {
		t.Fatal("email was not normalized")
	}
	var record database.EmailToken
	database.DB.First(&record)
	if record.CredentialHash == bindingToken || record.CredentialHash != hashVerificationToken(bindingToken) {
		t.Fatal("raw credential persisted")
	}
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(bindingToken), nil)
	requireEmailCode(t, response, 408)
	other, _ := performJSONRequest(r, "POST", "/login", `{"username":"bob","password":"old-password"}`, nil)
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(bindingToken), other.Result().Cookies())
	requireEmailCode(t, response, 400)
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(bindingToken), cookies)
	requireEmailCode(t, response, 200)
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(bindingToken), cookies)
	requireEmailCode(t, response, 400)
	clearEmailLimits(t)
	_, response = performJSONRequest(r, "POST", "/bind", `{"email":"alice@example.test","password":"old-password"}`, other.Result().Cookies())
	requireEmailCode(t, response, 409)
	clearEmailLimits(t)
	_, unknown := performJSONRequest(r, "POST", "/reset", `{"email":"unknown@example.test"}`, nil)
	requireEmailCode(t, unknown, 200)
	if len(sender.messages) != 1 {
		t.Fatal("sent mail to unknown account")
	}
	_, response = performJSONRequest(r, "POST", "/reset", `{"email":"alice@example.test"}`, nil)
	requireEmailCode(t, response, 200)
	if string(unknown.Data) != string(response.Data) {
		t.Fatal("reset response disclosed account existence")
	}
	resetToken := sender.token(t)
	_, response = performJSONRequest(r, "POST", "/reset", `{"email":"alice@example.test"}`, nil)
	requireEmailCode(t, response, 429)
	// Purpose isolation and invalid passwords must not consume the valid link.
	_, response = performJSONRequest(r, "POST", "/reset/confirm", resetBody(bindingToken, "new-password"), nil)
	requireEmailCode(t, response, 400)
	_, response = performJSONRequest(r, "POST", "/reset/confirm", resetBody(resetToken, "short"), nil)
	requireEmailCode(t, response, 400)
	clearEmailLimits(t)
	_, response = performJSONRequest(r, "POST", "/reset", `{"email":"alice@example.test"}`, nil)
	requireEmailCode(t, response, 200)
	secondToken := sender.token(t)
	_, response = performJSONRequest(r, "POST", "/reset/confirm", resetBody(resetToken, "new-password"), nil)
	requireEmailCode(t, response, 200)
	_, response = performJSONRequest(r, "POST", "/reset/confirm", resetBody(resetToken, "another-password"), nil)
	requireEmailCode(t, response, 400)
	_, response = performJSONRequest(r, "POST", "/reset/confirm", resetBody(secondToken, "another-password"), nil)
	requireEmailCode(t, response, 400)
	_, response = performJSONRequest(r, "GET", "/info", "", cookies)
	requireEmailCode(t, response, 408)
	_, response = performJSONRequest(r, "POST", "/login", `{"username":"alice","password":"old-password"}`, nil)
	requireEmailCode(t, response, 401)
	login, response := performJSONRequest(r, "POST", "/login", `{"username":"alice","password":"new-password"}`, nil)
	requireEmailCode(t, response, 200)
	_, response = performJSONRequest(r, "GET", "/info", "", login.Result().Cookies())
	requireEmailCode(t, response, 200)
}

func TestEmailFeatureDisabledAndExpiredLinks(t *testing.T) {
	r, sender, cookies := setupEmailTest(t)
	database.DB.Model(&database.Config{}).Where("1 = 1").Update("email_enabled", false)
	for _, path := range []string{"/bind", "/bind/confirm", "/reset", "/reset/confirm"} {
		_, response := performJSONRequest(r, "POST", path, `{}`, cookies)
		requireEmailCode(t, response, 404)
	}
	if len(sender.messages) != 0 {
		t.Fatal("mail sent when disabled")
	}
	database.DB.Model(&database.Config{}).Where("1 = 1").Update("email_enabled", true)
	_, response := performJSONRequest(r, "POST", "/bind", `{"email":"alice@example.test","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 200)
	token := sender.token(t)
	database.DB.Model(&database.EmailToken{}).Where("1 = 1").Update("expires_at", time.Now().UTC().Add(-time.Minute))
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(token), cookies)
	requireEmailCode(t, response, 400)
	email := "alice@example.test"
	database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 1).Update("email", email)
	clearEmailLimits(t)
	_, response = performJSONRequest(r, "POST", "/reset", `{"email":"alice@example.test"}`, nil)
	requireEmailCode(t, response, 200)
	token = sender.token(t)
	database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 1).Update("is_disabled", true)
	_, response = performJSONRequest(r, "POST", "/reset/confirm", resetBody(token, "new-password"), nil)
	requireEmailCode(t, response, 400)
	clearEmailLimits(t)
	before := len(sender.messages)
	_, response = performJSONRequest(r, "POST", "/reset", `{"email":"alice@example.test"}`, nil)
	requireEmailCode(t, response, 200)
	if len(sender.messages) != before {
		t.Fatal("sent reset email to disabled account")
	}
}

func TestEmailDeliveryFailureAndAddressValidation(t *testing.T) {
	r, sender, cookies := setupEmailTest(t)
	for _, address := range []string{"invalid", "Name <alice@example.test>", "alice@example.test\r\nBcc: victim@example.test"} {
		body, _ := json.Marshal(gin.H{"email": address, "password": "old-password"})
		_, response := performJSONRequest(r, "POST", "/bind", string(body), cookies)
		requireEmailCode(t, response, 400)
	}
	sender.err = errors.New("delivery rejected")
	_, response := performJSONRequest(r, "POST", "/bind", `{"email":"alice@example.test","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 502)
	var count int64
	database.DB.Model(&database.EmailToken{}).Count(&count)
	if count != 0 {
		t.Fatal("failed delivery left a live credential")
	}
	email := "alice@example.test"
	database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 1).Update("email", email)
	clearEmailLimits(t)
	_, response = performJSONRequest(r, "POST", "/reset", `{"email":"alice@example.test"}`, nil)
	requireEmailCode(t, response, 200)
	database.DB.Model(&database.EmailToken{}).Count(&count)
	if count != 0 {
		t.Fatal("failed reset delivery left a live credential")
	}
}

func TestSMTPSettingsValidationAndPrivacy(t *testing.T) {
	r, _, _ := setupEmailTest(t)
	_, response := performJSONRequest(r, "GET", "/config", "", nil)
	if strings.Contains(string(response.Data), "smtp") || strings.Contains(string(response.Data), "private-password") {
		t.Fatal("SMTP settings exposed publicly")
	}
	_, response = performJSONRequest(r, "GET", "/settings", "", nil)
	var settings SettingsRequest
	if err := json.Unmarshal(response.Data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Email.Password != "" || !settings.Email.PasswordConfigured {
		t.Fatal("SMTP password was returned")
	}
	body, _ := json.Marshal(settings)
	_, response = performJSONRequest(r, "PUT", "/settings", string(body), nil)
	requireEmailCode(t, response, 200)
	var config database.Config
	database.DB.First(&config)
	if config.SMTPPassword != "private-password" {
		t.Fatal("blank password overwrote stored password")
	}
	settings.Email.SiteURL = "https://ask.example.test/?redirect=evil"
	body, _ = json.Marshal(settings)
	_, response = performJSONRequest(r, "PUT", "/settings", string(body), nil)
	requireEmailCode(t, response, 400)
	settings.Email.SiteURL = "https://ask.example.test"
	settings.Email.Port = 65536
	body, _ = json.Marshal(settings)
	_, response = performJSONRequest(r, "PUT", "/settings", string(body), nil)
	requireEmailCode(t, response, 400)
	settings.Email.Enabled = false
	body, _ = json.Marshal(settings)
	_, response = performJSONRequest(r, "PUT", "/settings", string(body), nil)
	requireEmailCode(t, response, 200)
}

func TestEmailRateLimitUnknownAddresses(t *testing.T) {
	r, _, _ := setupEmailTest(t)
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(gin.H{"email": string(rune('a'+i)) + "@example.test"})
		_, response := performJSONRequest(r, "POST", "/reset", string(body), nil)
		requireEmailCode(t, response, 200)
	}
	_, response := performJSONRequest(r, "POST", "/reset", `{"email":"six@example.test"}`, nil)
	requireEmailCode(t, response, 429)
}

func TestBoundEmailCannotBeChanged(t *testing.T) {
	r, sender, cookies := setupEmailTest(t)
	// Two links may be issued before the first binding is confirmed.
	_, response := performJSONRequest(r, "POST", "/bind", `{"email":"first@example.test","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 200)
	firstToken := sender.token(t)
	_, response = performJSONRequest(r, "POST", "/bind", `{"email":"second@example.test","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 200)
	secondToken := sender.token(t)
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(firstToken), cookies)
	requireEmailCode(t, response, 200)
	count := len(sender.messages)
	_, response = performJSONRequest(r, "POST", "/bind", `{"email":"third@example.test","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 409)
	if len(sender.messages) != count {
		t.Fatal("sent email for unsupported replacement")
	}
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(secondToken), cookies)
	requireEmailCode(t, response, 400)
	// A replacement link issued by an older backend must also be rejected.
	token, hash, err := newVerificationToken()
	if err != nil {
		t.Fatal(err)
	}
	var user database.User
	database.DB.First(&user, 1)
	oldLink := database.EmailToken{BilibiliUID: 1, Purpose: emailPurposeBind, Email: "third@example.test", PreviousEmail: "first@example.test", PasswordHash: user.PasswordHash, CredentialHash: hash, ExpiresAt: time.Now().UTC().Add(time.Minute)}
	if err := database.DB.Create(&oldLink).Error; err != nil {
		t.Fatal(err)
	}
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(token), cookies)
	requireEmailCode(t, response, 400)
	user = database.User{}
	database.DB.First(&user, 1)
	if user.Email == nil || *user.Email != "first@example.test" {
		t.Fatal("bound email changed")
	}
}

func TestBindingEmailUniquenessAtConfirmation(t *testing.T) {
	r, sender, cookies := setupEmailTest(t)
	_, response := performJSONRequest(r, "POST", "/bind", `{"email":"shared@example.test","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 200)
	bindingToken := sender.token(t)
	// Another account claims the address after the mail was issued.
	if err := database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 2).Update("email", "shared@example.test").Error; err != nil {
		t.Fatal(err)
	}
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(bindingToken), cookies)
	requireEmailCode(t, response, 400)
	var user database.User
	database.DB.First(&user, 1)
	if user.Email != nil {
		t.Fatal("duplicate email binding was accepted")
	}
	var token database.EmailToken
	database.DB.Where("credential_hash = ?", hashVerificationToken(bindingToken)).First(&token)
	if token.ConsumedAt != nil {
		t.Fatal("failed transaction consumed the link")
	}
}

func TestLoginWithUsernameOrBoundEmail(t *testing.T) {
	r, _, _ := setupEmailTest(t)
	email := "an.email.address.longer.than.thirty.two.characters@example.test"
	if err := database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 1).Update("email", email).Error; err != nil {
		t.Fatal(err)
	}
	for _, identifier := range []string{"alice", email, " " + strings.ToUpper(email) + " "} {
		body, _ := json.Marshal(gin.H{"username": identifier, "password": "old-password"})
		recorder, response := performJSONRequest(r, "POST", "/login", string(body), nil)
		requireEmailCode(t, response, 200)
		var user struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(response.Data, &user); err != nil {
			t.Fatal(err)
		}
		if user.Username != "alice" {
			t.Fatal("authenticated wrong account")
		}
		_, response = performJSONRequest(r, "GET", "/info", "", recorder.Result().Cookies())
		requireEmailCode(t, response, 200)
	}
	for _, identifier := range []string{"", "unknown@example.test", "invalid@email", strings.Repeat("x", 255)} {
		body, _ := json.Marshal(gin.H{"username": identifier, "password": "old-password"})
		_, response := performJSONRequest(r, "POST", "/login", string(body), nil)
		requireEmailCode(t, response, 401)
	}
	body, _ := json.Marshal(gin.H{"username": email, "password": "wrong-password"})
	_, response := performJSONRequest(r, "POST", "/login", string(body), nil)
	requireEmailCode(t, response, 401)
	// SMTP availability does not affect existing login credentials.
	database.DB.Model(&database.Config{}).Where("1 = 1").Update("email_enabled", false)
	body, _ = json.Marshal(gin.H{"username": email, "password": "old-password"})
	_, response = performJSONRequest(r, "POST", "/login", string(body), nil)
	requireEmailCode(t, response, 200)
	database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 1).Update("is_disabled", true)
	_, response = performJSONRequest(r, "POST", "/login", string(body), nil)
	requireEmailCode(t, response, 403)
}

func TestUnconfirmedEmailCannotLogin(t *testing.T) {
	r, sender, cookies := setupEmailTest(t)
	_, response := performJSONRequest(r, "POST", "/bind", `{"email":"alice@example.test","password":"old-password"}`, cookies)
	requireEmailCode(t, response, 200)
	_, response = performJSONRequest(r, "POST", "/login", `{"username":"alice@example.test","password":"old-password"}`, nil)
	requireEmailCode(t, response, 401)
	_, response = performJSONRequest(r, "POST", "/bind/confirm", confirmBody(sender.token(t)), cookies)
	requireEmailCode(t, response, 200)
	_, response = performJSONRequest(r, "POST", "/login", `{"username":"alice@example.test","password":"old-password"}`, nil)
	requireEmailCode(t, response, 200)
}

func TestEmailShapedUsernameCollision(t *testing.T) {
	r, _, _ := setupEmailTest(t)
	email := "alice@example.test"
	if err := database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 1).Update("email", email).Error; err != nil {
		t.Fatal(err)
	}
	bobHash, err := bcrypt.GenerateFromPassword([]byte("bob-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 2).Updates(map[string]any{"username": email, "password_hash": string(bobHash)}).Error; err != nil {
		t.Fatal(err)
	}
	for password, expected := range map[string]string{"old-password": "alice", "bob-password": email} {
		body, _ := json.Marshal(gin.H{"username": email, "password": password})
		_, response := performJSONRequest(r, "POST", "/login", string(body), nil)
		requireEmailCode(t, response, 200)
		var user struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(response.Data, &user); err != nil {
			t.Fatal(err)
		}
		if user.Username != expected {
			t.Fatal("namespace collision selected wrong account")
		}
	}
	var alice database.User
	database.DB.First(&alice, 1)
	database.DB.Model(&database.User{}).Where("bilibili_uid = ?", 2).Update("password_hash", alice.PasswordHash)
	_, response := performJSONRequest(r, "POST", "/login", `{"username":"alice@example.test","password":"old-password"}`, nil)
	requireEmailCode(t, response, 401)
}
