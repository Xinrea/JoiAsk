package database

import (
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
	TagID     int    `gorm:"index" json:"tag_id"`
	Tag       Tag    `gorm:"foreignkey:TagID" json:"tag"`
	Content   string `json:"content"`
	ImagesNum int    `json:"images_num"`
	Images    string `json:"images"`
	Likes     int    `json:"likes"`
	IsHide    bool   `gorm:"index" json:"is_hide"`
	IsRainbow bool   `gorm:"index" json:"is_rainbow"`
	IsArchive bool   `gorm:"index" json:"is_archive"`
	IsPublish bool   `gorm:"index" json:"is_publish"`
}

type LikeRecord struct {
	BaseModel
	IP         string   `gorm:"index" json:"ip"`
	QuestionID int      `gorm:"index" json:"question_id"`
	Question   Question `json:"question"`
}

type Admin struct {
	BaseModel
	Username string `gorm:"unique" json:"username"`
	Password string `json:"-"`
}

type Config struct {
	BaseModel
	Announcement string `json:"announcement"`
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
