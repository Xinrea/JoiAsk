package database

import (
	"encoding/json"
	"strconv"
	"time"
)

type BaseModel struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}
type Tag struct {
	BaseModel
	TagName     string `gorm:"unique" json:"tag_name"`
	Description string `json:"description"`
}

type Question struct {
	BaseModel
	BilibiliUID    *int64 `gorm:"index" json:"-"`
	TagID          int    `gorm:"index" json:"tag_id"`
	Tag            Tag    `gorm:"foreignkey:TagID" json:"tag"`
	Content        string `json:"content"`
	ImagesNum      int    `json:"images_num"`
	Images         string `json:"images"`
	Likes          int    `json:"likes"`
	IsHide         bool   `gorm:"index" json:"is_hide"`
	IsRainbow      bool   `gorm:"index" json:"is_rainbow"`
	IsArchive      bool   `gorm:"index" json:"is_archive"`
	IsPublish      bool   `gorm:"index" json:"is_publish"`
	IsSpam         bool   `gorm:"index;not null;default:false" json:"is_spam"`
	IsRealName     bool   `gorm:"index;not null;default:false" json:"is_real_name"`
	BilibiliName   string `gorm:"size:255" json:"bilibili_name,omitempty"`
	BilibiliAvatar string `gorm:"size:1024" json:"bilibili_avatar,omitempty"`
	Emojis         string `json:"emojis"`
}

func (q Question) MarshalJSON() ([]byte, error) {
	type questionAlias Question
	copy := q
	var publicUID *string
	if copy.IsRealName && copy.BilibiliUID != nil {
		uid := strconv.FormatInt(*copy.BilibiliUID, 10)
		publicUID = &uid
	} else {
		copy.BilibiliName = ""
		copy.BilibiliAvatar = ""
	}
	return json.Marshal(struct {
		*questionAlias
		BilibiliUID *string `json:"bilibili_uid,omitempty"`
	}{
		questionAlias: (*questionAlias)(&copy),
		BilibiliUID:   publicUID,
	})
}

type User struct {
	BilibiliUID    int64     `gorm:"primaryKey;autoIncrement:false" json:"bilibili_uid"`
	Username       string    `gorm:"size:32;uniqueIndex;not null" json:"username"`
	PasswordHash   string    `gorm:"size:255;not null" json:"-"`
	BilibiliName   string    `gorm:"size:255;not null" json:"bilibili_name"`
	BilibiliAvatar string    `gorm:"size:1024;not null" json:"bilibili_avatar"`
	VerifiedAt     time.Time `gorm:"not null" json:"verified_at"`
	IsDisabled     bool      `gorm:"index;not null;default:false" json:"is_disabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BilibiliVerificationAccount struct {
	BaseModel
	BilibiliUID      int64      `gorm:"uniqueIndex;not null" json:"bilibili_uid"`
	EncryptedCookie  string     `gorm:"type:text;not null" json:"-"`
	LastCheckedAt    *time.Time `json:"last_checked_at"`
	LastSuccessfulAt *time.Time `json:"last_successful_at"`
	LastError        string     `gorm:"type:text" json:"last_error"`
}

const (
	VerificationPending  = "pending"
	VerificationVerified = "verified"
	VerificationExpired  = "expired"
	VerificationConsumed = "consumed"
)

type BilibiliVerificationRequest struct {
	BaseModel
	BilibiliUID       int64      `gorm:"index;not null" json:"bilibili_uid"`
	CredentialHash    string     `gorm:"size:64;not null" json:"-"`
	Status            string     `gorm:"size:16;index;not null" json:"status"`
	RequestedAt       time.Time  `gorm:"index;not null" json:"requested_at"`
	ExpiresAt         time.Time  `gorm:"index;not null" json:"expires_at"`
	FollowedAt        *time.Time `json:"followed_at"`
	BilibiliName      string     `gorm:"size:255" json:"bilibili_name"`
	BilibiliAvatar    string     `gorm:"size:1024" json:"bilibili_avatar"`
	VerifiedAt        *time.Time `json:"verified_at"`
	ConfirmationUntil *time.Time `gorm:"index" json:"confirmation_until"`
	ConsumedAt        *time.Time `json:"consumed_at"`
}

type LikeRecord struct {
	BaseModel
	IP         string   `gorm:"uniqueIndex:idx_like_records_ip_question" json:"ip"`
	QuestionID int      `gorm:"uniqueIndex:idx_like_records_ip_question" json:"question_id"`
	Question   Question `json:"question"`
}

type Admin struct {
	BaseModel
	Username string `gorm:"unique" json:"username"`
	Password string `json:"-"`
}

type Config struct {
	BaseModel
	Announcement              string `json:"announcement"`
	RequireVerifiedUserToPost bool   `gorm:"not null;default:false" json:"require_verified_user_to_post"`
	DeepSeekAPIKey            string `json:"-"`
	SpamPrompt                string `gorm:"type:text" json:"-"`
}

func (t Tag) Json() map[string]interface{} {
	var count int64
	DB.Model(&Question{}).Where("tag_id = ?", t.ID).Count(&count)
	return map[string]interface{}{
		"id":             t.ID,
		"tag_name":       t.TagName,
		"description":    t.Description,
		"question_count": count,
		"created_at":     t.CreatedAt,
	}
}
