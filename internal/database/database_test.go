package database

import (
	"encoding/json"
	"joiask-backend/internal/deepseek"
	"joiask-backend/pkg/util"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
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

	var admin Admin
	if err := DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("admin")) != nil {
		t.Fatal("new default administrator must use bcrypt")
	}

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
	if config.EmailEnabled {
		t.Fatal("email must default to disabled after migration")
	}
	if !DB.Migrator().HasTable(&EmailToken{}) || !DB.Migrator().HasTable(&EmailRateLimit{}) {
		t.Fatal("email tables were not migrated")
	}
	if config.SpamPrompt != deepseek.DefaultSpamPrompt {
		t.Fatalf("unexpected default spam prompt: %q", config.SpamPrompt)
	}

	var emojiCount int64
	if err := DB.Model(&Emoji{}).Count(&emojiCount).Error; err != nil {
		t.Fatal(err)
	}
	if emojiCount != int64(len(DefaultEmojis)) {
		t.Fatalf("expected %d default emojis, got %d", len(DefaultEmojis), emojiCount)
	}
	if err := DB.Where("tag = ?", DefaultEmojis[0].Tag).Delete(&Emoji{}).Error; err != nil {
		t.Fatal(err)
	}
	initializeDB()
	if err := DB.Model(&Emoji{}).Count(&emojiCount).Error; err != nil {
		t.Fatal(err)
	}
	if emojiCount != int64(len(DefaultEmojis)-1) {
		t.Fatalf("expected deleted default emoji to stay deleted, got %d records", emojiCount)
	}
}

func TestInitializeDBPreservesLegacyAdminPassword(t *testing.T) {
	var err error
	DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-admin.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&Admin{}); err != nil {
		t.Fatal(err)
	}
	legacy := util.Md5v("existing-password")
	if err := DB.Create(&Admin{Username: "admin", Password: legacy}).Error; err != nil {
		t.Fatal(err)
	}
	initializeDB()
	initializeDB()
	var admin Admin
	if err := DB.Where("username = ?", "admin").First(&admin).Error; err != nil || admin.Password != legacy {
		t.Fatal("initialization must preserve existing admin credentials")
	}
}

type adminBeforeSessionVersion struct {
	BaseModel
	Username string `gorm:"unique"`
	Password string
}

func (adminBeforeSessionVersion) TableName() string { return "admins" }

func TestInitializeDBMigratesAdminSessionVersion(t *testing.T) {
	previous := DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "admin-session-version.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	DB = db
	t.Cleanup(func() {
		DB = previous
		sqlDB.Close()
	})
	if err := DB.AutoMigrate(&adminBeforeSessionVersion{}); err != nil {
		t.Fatal(err)
	}
	legacy := adminBeforeSessionVersion{Username: "admin", Password: "existing-hash"}
	if err := DB.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	initializeDB()
	var admin Admin
	if err := DB.First(&admin, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if admin.Password != legacy.Password || admin.SessionVersion != 0 {
		t.Fatal("migration changed existing credential or failed to initialize version")
	}
	if err := DB.Model(&admin).Update("session_version", 7).Error; err != nil {
		t.Fatal(err)
	}
	initializeDB()
	if err := DB.First(&admin, legacy.ID).Error; err != nil || admin.SessionVersion != 7 {
		t.Fatal("repeated initialization reset the session version")
	}
	data, err := json.Marshal(admin)
	if err != nil || strings.Contains(string(data), "session_version") || strings.Contains(string(data), "existing-hash") {
		t.Fatalf("administrator JSON exposed credentials or version: %s, %v", data, err)
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

type userBeforeEmail struct {
	BilibiliUID    int64     `gorm:"primaryKey;autoIncrement:false"`
	Username       string    `gorm:"size:32;uniqueIndex;not null"`
	PasswordHash   string    `gorm:"size:255;not null"`
	BilibiliName   string    `gorm:"size:255;not null"`
	BilibiliAvatar string    `gorm:"size:1024;not null"`
	VerifiedAt     time.Time `gorm:"not null"`
	IsDisabled     bool      `gorm:"index;not null;default:false"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (userBeforeEmail) TableName() string { return "users" }
func TestEmailMigrationPreservesExistingUsers(t *testing.T) {
	var err error
	DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "email-migration.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&userBeforeEmail{}); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"alice", "bob"} {
		if err := DB.Create(&userBeforeEmail{BilibiliUID: int64(i + 1), Username: name, PasswordHash: "existing-hash", VerifiedAt: time.Now()}).Error; err != nil {
			t.Fatal(err)
		}
	}
	initializeDB()
	var users []User
	if err := DB.Order("bilibili_uid").Find(&users).Error; err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatal("migration lost existing users")
	}
	for _, user := range users {
		if user.Email != nil || user.SessionVersion != 0 || user.PasswordHash != "existing-hash" {
			t.Fatal("migration changed existing account credentials")
		}
	}
}
