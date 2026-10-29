package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"joiask-backend/internal/database"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRecordIDTestDB(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previous := database.DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ids.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(&database.Config{}, &database.Tag{}, &database.Question{}, &database.Admin{}, &database.Emoji{}, &database.LikeRecord{}, &database.User{}, &database.BilibiliVerificationRequest{}); err != nil {
		t.Fatal(err)
	}
	for _, record := range []interface{}{
		&database.Config{},
		&database.Tag{BaseModel: database.BaseModel{ID: 1}, TagName: "default"},
		&database.Tag{BaseModel: database.BaseModel{ID: 2}, TagName: "other"},
		&database.Question{BaseModel: database.BaseModel{ID: 2}, TagID: 1, Content: "original"},
		&database.Admin{BaseModel: database.BaseModel{ID: 1}, Username: "operator", Password: "test-only", Role: database.AdminRoleSuperAdmin},
		&database.Admin{BaseModel: database.BaseModel{ID: 2}, Username: "reviewer", Password: "test-only", Role: database.AdminRoleAdmin},
		&database.Emoji{BaseModel: database.BaseModel{ID: 2}, Tag: "[test]", URL: "/test.png"},
	} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
}

type recordIDEndpoint struct {
	name    string
	handler gin.HandlerFunc
	body    string
	form    bool
	model   interface{}
}

func recordIDEndpoints() []recordIDEndpoint {
	question := &QuestionController{eventChan: make(chan SSEvent, 1)}
	return []recordIDEndpoint{
		{"question/post", question.Post, "", true, &database.Question{}},
		{"question/put", question.Put, `{"tag_id":1,"is_publish":true}`, false, &database.Question{}},
		{"question/delete", question.Delete, "", false, &database.Question{}},
		{"tag/put", (&TagController{}).Put, `{"tag_name":"updated","description":"updated"}`, false, &database.Tag{}},
		{"tag/delete", (&TagController{}).Delete, "", false, &database.Tag{}},
		{"emoji/put", (&EmojiController{}).Put, `{"tag":"[updated]","url":"/updated.png"}`, false, &database.Emoji{}},
		{"emoji/delete", (&EmojiController{}).Delete, "", false, &database.Emoji{}},
		{"user/put", (&UserController{}).Put, `{"username":"updated","password":"updated"}`, false, &database.Admin{}},
		{"user/delete", (&UserController{}).Delete, "", false, &database.Admin{}},
	}
}

func requestRecordID(t *testing.T, endpoint recordIDEndpoint, raw string) testAPIResponse {
	t.Helper()
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-session-secret-that-is-long-enough"))))
	router.POST("/test", func(c *gin.Context) {
		// Exercise the handler after router parameter extraction and admin auth.
		c.Params = gin.Params{{Key: "id", Value: raw}}
		c.Set("user", database.Admin{BaseModel: database.BaseModel{ID: 1}, Username: "operator", Role: database.AdminRoleSuperAdmin})
		endpoint.handler(c)
	})
	var body bytes.Buffer
	contentType := "application/json"
	if endpoint.form {
		writer := multipart.NewWriter(&body)
		for key, value := range map[string]string{"tag_id": raw, "content": "test question"} {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		contentType = writer.FormDataContentType()
	} else {
		body.WriteString(endpoint.body)
	}
	request := httptest.NewRequest(http.MethodPost, "/test", &body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var response testAPIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response (HTTP %d): %s: %v", recorder.Code, recorder.Body.String(), err)
	}
	return response
}

func TestRecordIDEndpointsRejectInvalidInputBeforeQuery(t *testing.T) {
	setupRecordIDTestDB(t)
	queries, writes := 0, 0
	if err := database.DB.Callback().Query().Before("gorm:query").Register("test:count_id_queries", func(tx *gorm.DB) {
		// Posting reads configuration before checking the ID.
		if tx.Statement.Table != "configs" {
			queries++
		}
	}); err != nil {
		t.Fatal(err)
	}
	countWrite := func(*gorm.DB) { writes++ }
	for _, err := range []error{
		database.DB.Callback().Create().Before("gorm:create").Register("test:count_id_writes", countWrite),
		database.DB.Callback().Update().Before("gorm:update").Register("test:count_id_writes", countWrite),
		database.DB.Callback().Delete().Before("gorm:delete").Register("test:count_id_writes", countWrite),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	invalid := []string{
		"", "0", "-1", "+1", " 1", "1 ", "1.0", "0x1", "１", "abc",
		"9223372036854775808", "18446744073709551616", "1 AND 1=1", "1 AND 1=2", "1 OR 1=1",
		"1 AND (SELECT substr('sample',1,1))='s'", "1; SELECT 1", "1--",
	}
	for _, endpoint := range recordIDEndpoints() {
		t.Run(endpoint.name, func(t *testing.T) {
			for _, raw := range invalid {
				t.Run(fmt.Sprintf("%q", raw), func(t *testing.T) {
					queries, writes = 0, 0
					response := requestRecordID(t, endpoint, raw)
					if response.Code != 400 || queries != 0 || writes != 0 {
						t.Fatalf("input %q: response=%+v queries=%d writes=%d", raw, response, queries, writes)
					}
				})
			}
		})
	}
}

func TestRecordIDEndpointsPreserveNotFound(t *testing.T) {
	setupRecordIDTestDB(t)
	for _, endpoint := range recordIDEndpoints() {
		t.Run(endpoint.name, func(t *testing.T) {
			if response := requestRecordID(t, endpoint, "999"); response.Code != 404 {
				t.Fatalf("unexpected response: %+v", response)
			}
		})
	}
}

func TestRecordIDEndpointsAcceptNumericIDs(t *testing.T) {
	for _, endpoint := range recordIDEndpoints() {
		t.Run(endpoint.name, func(t *testing.T) {
			setupRecordIDTestDB(t)
			if endpoint.form {
				// The post handler launches a spam check. Wait for its final DB read
				// before restoring the global DB; no API key means no external call.
				done := make(chan struct{})
				reads := 0
				if err := database.DB.Callback().Query().After("gorm:query").Register("test:spam_config_read", func(tx *gorm.DB) {
					if tx.Statement.Table == "configs" {
						reads++
						if reads == 2 {
							close(done)
						}
					}
				}); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					select {
					case <-done:
					case <-time.After(5 * time.Second):
						t.Error("spam check did not finish its database read")
					}
				})
			}
			if response := requestRecordID(t, endpoint, "2"); response.Code != 200 {
				t.Fatalf("unexpected response: %+v", response)
			}
			if strings.HasSuffix(endpoint.name, "/delete") {
				var count int64
				if err := database.DB.Model(endpoint.model).Where("id = ?", 2).Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("record not deleted: count=%d error=%v", count, err)
				}
			} else if strings.HasSuffix(endpoint.name, "/put") {
				if err := database.DB.First(endpoint.model, 2).Error; err != nil {
					t.Fatal(err)
				}
				updated := false
				switch record := endpoint.model.(type) {
				case *database.Question:
					updated = record.IsPublish && record.TagID == 1
				case *database.Tag:
					updated = record.TagName == "updated" && record.Description == "updated"
				case *database.Emoji:
					updated = record.Tag == "[updated]" && record.URL == "/updated.png"
				case *database.Admin:
					updated = record.Username == "updated" && bcrypt.CompareHashAndPassword([]byte(record.Password), []byte("updated")) == nil
				}
				if !updated {
					t.Fatalf("record not updated: %+v", endpoint.model)
				}
			} else if endpoint.form {
				var question database.Question
				if err := database.DB.Where("tag_id = ?", 2).First(&question).Error; err != nil || question.Content != "test question" {
					t.Fatalf("question not created: %+v error=%v", question, err)
				}
			}
		})
	}
}

func TestQuestionPostRejectsURLEncodedInjection(t *testing.T) {
	setupRecordIDTestDB(t)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-session-secret-that-is-long-enough"))))
	router.POST("/question", (&QuestionController{}).Post)
	for _, raw := range []string{"1 AND 1=1", "1 AND 1=2", ""} {
		body := url.Values{"tag_id": {raw}}.Encode()
		request := httptest.NewRequest(http.MethodPost, "/question", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		var response testAPIResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || response.Code != 400 {
			t.Fatalf("input %q: response=%s error=%v", raw, recorder.Body.String(), err)
		}
	}
}

func TestSessionRecordIDsRequireStoredNumericTypes(t *testing.T) {
	setupRecordIDTestDB(t)
	for _, record := range []interface{}{
		&database.User{BilibiliUID: 12, Username: "member", PasswordHash: "test-only"},
		&database.BilibiliVerificationRequest{BaseModel: database.BaseModel{ID: 12}, CredentialHash: hashVerificationToken("test-token")},
	} {
		if err := database.DB.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	queries := 0
	if err := database.DB.Callback().Query().Before("gorm:query").Register("test:session_queries", func(*gorm.DB) { queries++ }); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name         string
		verification interface{}
		member       interface{}
		valid        bool
	}{
		{"valid", uint(12), int64(12), true},
		{"missing", nil, nil, false},
		{"zero", uint(0), int64(0), false},
		{"wrong types", int64(12), uint(12), false},
		{"negative", int(-1), int64(-1), false},
		{"numeric strings", "12", "12", false},
		{"SQL strings", "1 OR 1=1", "1 OR 1=1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-session-secret-that-is-long-enough"))))
			router.POST("/set", func(c *gin.Context) {
				session := sessions.Default(c)
				if tc.verification != nil {
					session.Set(verificationRequestSessionKey, tc.verification)
				}
				if tc.member != nil {
					session.Set(memberSessionKey, tc.member)
				}
				session.Set(verificationTokenSessionKey, "test-token")
				if err := session.Save(); err != nil {
					t.Fatal(err)
				}
				Success(c, nil)
			})
			router.POST("/check", func(c *gin.Context) {
				_, err := loadSessionVerification(c)
				_, member := currentMember(c)
				if (err == nil) != tc.valid || member != tc.valid {
					t.Errorf("unexpected session result: verification error=%v member=%t", err, member)
				}
				Success(c, nil)
			})
			set, _ := performJSONRequest(router, http.MethodPost, "/set", "", nil)
			queries = 0
			performJSONRequest(router, http.MethodPost, "/check", "", set.Result().Cookies())
			wantQueries := 0
			if tc.valid {
				wantQueries = 2
			}
			if queries != wantQueries {
				t.Errorf("got %d queries, want %d", queries, wantQueries)
			}
		})
	}
}

func TestParseRecordID(t *testing.T) {
	for _, raw := range []string{"1", "001", strconv.Itoa(int(^uint(0) >> 1))} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		want, _ := strconv.Atoi(raw)
		if id, ok := parseRecordID(c, raw); !ok || id != want {
			t.Errorf("parseRecordID(%q) = %d, %t; want %d", raw, id, ok, want)
		}
	}
}
