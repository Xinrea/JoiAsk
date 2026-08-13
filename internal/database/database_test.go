package database

import (
	"joiask-backend/internal/deepseek"
	"path/filepath"
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
