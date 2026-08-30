package controller

import (
	"encoding/json"
	"joiask-backend/internal/database"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestQuestionsWithIdentityEnrichesAnonymousSubmitter(t *testing.T) {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "question.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.AutoMigrate(&database.User{}); err != nil {
		t.Fatal(err)
	}

	uid := int64(123456)
	if err := database.DB.Create(&database.User{
		BilibiliUID:    uid,
		Username:       "member",
		PasswordHash:   "hash",
		BilibiliName:   "测试用户",
		BilibiliAvatar: "/upload-img/avatar.jpg",
	}).Error; err != nil {
		t.Fatal(err)
	}

	response, err := json.Marshal(questionsWithIdentity([]database.Question{{
		BilibiliUID: &uid,
		IsRealName:  false,
	}}))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`"bilibili_uid":"123456"`,
		`"bilibili_name":"测试用户"`,
		`"bilibili_avatar":"/upload-img/avatar.jpg"`,
	} {
		if !strings.Contains(string(response), expected) {
			t.Fatalf("admin question missing %s: %s", expected, response)
		}
	}
}
