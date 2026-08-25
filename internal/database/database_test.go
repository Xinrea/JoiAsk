package database

import (
	"encoding/json"
	"joiask-backend/internal/deepseek"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type previousQuestion struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	TagID     int
	Content   string
	ImagesNum int
	Images    string
	Likes     int
	IsHide    bool
	IsRainbow bool
	IsArchive bool
	IsPublish bool
	Emojis    string
}

func (previousQuestion) TableName() string { return "questions" }

type previousConfig struct {
	ID           uint `gorm:"primary_key"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Announcement string
}

func (previousConfig) TableName() string { return "configs" }

func TestInitializeDBFromPreviousSchema(t *testing.T) {
	var err error
	DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ask.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := DB.AutoMigrate(&previousQuestion{}, &previousConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&previousQuestion{Content: "existing question"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&previousConfig{Announcement: "existing announcement"}).Error; err != nil {
		t.Fatal(err)
	}

	initializeDB()

	var question Question
	if err := DB.First(&question).Error; err != nil {
		t.Fatal(err)
	}
	if question.IsSpam {
		t.Fatal("existing question should default to non-spam")
	}

	var nullSpamCount int64
	if err := DB.Model(&Question{}).Where("is_spam IS NULL").Count(&nullSpamCount).Error; err != nil {
		t.Fatal(err)
	}
	if nullSpamCount != 0 {
		t.Fatalf("expected no NULL is_spam values, got %d", nullSpamCount)
	}

	var config Config
	if err := DB.First(&config).Error; err != nil {
		t.Fatal(err)
	}
	if config.SpamPrompt != deepseek.DefaultSpamPrompt {
		t.Fatalf("unexpected default spam prompt: %q", config.SpamPrompt)
	}
}

func TestQuestionBilibiliUIDIsNotAForeignKey(t *testing.T) {
	var err error
	DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "foreign-key.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&User{}, &Question{}); err != nil {
		t.Fatal(err)
	}
	if DB.Migrator().HasConstraint(&Question{}, "fk_questions_bilibili_uid") {
		t.Fatal("questions.bilibili_uid must not have a foreign key")
	}
	user := User{Username: "123", PasswordHash: "hash", BilibiliUID: 123, BilibiliName: "test", VerifiedAt: time.Now()}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	uid := user.BilibiliUID
	if err := DB.Create(&Question{BilibiliUID: &uid}).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Delete(&user).Error; err != nil {
		t.Fatal(err)
	}
	var storedUID *int64
	if err := DB.Raw("SELECT bilibili_uid FROM questions LIMIT 1").Scan(&storedUID).Error; err != nil {
		t.Fatal(err)
	}
	if storedUID == nil || *storedUID != user.BilibiliUID {
		t.Fatalf("historical Bilibili UID was not preserved: %v", storedUID)
	}
}

func TestUserUsesBilibiliUIDAsPrimaryKey(t *testing.T) {
	var err error
	DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "user-primary-key.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}

	type columnInfo struct {
		Name string
		PK   int `gorm:"column:pk"`
	}
	var columns []columnInfo
	if err := DB.Raw("PRAGMA table_info(users)").Scan(&columns).Error; err != nil {
		t.Fatal(err)
	}
	primaryKey := ""
	for _, column := range columns {
		if column.Name == "id" {
			t.Fatal("users table must not contain an internal id column")
		}
		if column.PK > 0 {
			primaryKey = column.Name
		}
	}
	if primaryKey != "bilibili_uid" {
		t.Fatalf("expected bilibili_uid primary key, got %q", primaryKey)
	}
}

func TestQuestionOnlyExposesBilibiliIdentityForRealNamePosts(t *testing.T) {
	uid := int64(123456)
	anonymous, err := json.Marshal(Question{
		BilibiliUID:    &uid,
		BilibiliName:   "测试用户",
		BilibiliAvatar: "/upload-img/avatar.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"bilibili_uid", "username", "bilibili_name", "bilibili_avatar"} {
		if strings.Contains(string(anonymous), field) {
			t.Fatalf("anonymous question exposed %s: %s", field, anonymous)
		}
	}

	realName, err := json.Marshal(Question{
		BilibiliUID:    &uid,
		IsRealName:     true,
		BilibiliName:   "测试用户",
		BilibiliAvatar: "/upload-img/avatar.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"bilibili_uid":"123456"`, `"bilibili_name":"测试用户"`, `"bilibili_avatar":"/upload-img/avatar.jpg"`} {
		if !strings.Contains(string(realName), expected) {
			t.Fatalf("real-name question missing %s: %s", expected, realName)
		}
	}
}
