package database

import "gorm.io/gorm"

type Tag struct {
	gorm.Model
	TagName string
}

type Question struct {
	gorm.Model
	TagID       int `gorm:"index"`
	Content     string
	ImagesNum   int
	Images      string
	Likes       int
	IsHide      bool
	IsRainbow   bool `gorm:"index"`
	IsArchived  bool `gorm:"index"`
	IsPublished bool `gorm:"index"`
}

type LikeRecord struct {
	gorm.Model
	IP         string
	QuestionID int `gorm:"index"`
	Question   Question
}

type Admin struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string
}

type Config struct {
	gorm.Model
	Announcement string
}
