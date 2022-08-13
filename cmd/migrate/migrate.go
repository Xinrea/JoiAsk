package main

import (
	"fmt"
	"joiask-backend/internal/database"
	"joiask-backend/internal/database/oldmodels"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Only use for joi's question box
func main() {
	log.Info("Starting migration")
	viper.SetConfigFile("./config/config.json")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Config loaded")
	v1Dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", viper.GetString("mysql.user"), viper.GetString("mysql.pass"), viper.GetString("mysql.host"), viper.GetInt("mysql.port"), "jask")
	V1DB, err := gorm.Open(mysql.Open(v1Dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	v2Dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", viper.GetString("mysql.user"), viper.GetString("mysql.pass"), viper.GetString("mysql.host"), viper.GetInt("mysql.port"), "jask_v2")
	V2DB, err := gorm.Open(mysql.Open(v2Dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	// Tag Table
	var oldTags []oldmodels.Tag
	V1DB.Find(&oldTags)
	for _, tag := range oldTags {
		if tag.ID == 1 {
			continue
		}
		var newTag database.Tag
		newTag.ID = tag.ID
		newTag.CreatedAt = tag.CreatedAt
		newTag.UpdatedAt = tag.UpdatedAt
		newTag.TagName = tag.TagName
		newTag.Description = ""
		V2DB.Create(&newTag)
	}
	log.Info("Tag migration finished:", len(oldTags))
	// Question Table
	var oldQuestions []oldmodels.Question
	V1DB.Find(&oldQuestions)
	for _, question := range oldQuestions {
		var newQuestion database.Question
		newQuestion.ID = question.ID
		newQuestion.CreatedAt = question.CreatedAt
		newQuestion.UpdatedAt = question.UpdatedAt
		newQuestion.Content = question.Content
		newQuestion.Likes = question.Likes
		newQuestion.TagID = question.TagID
		newQuestion.ImagesNum = question.ImagesNum
		newQuestion.IsHide = question.IsHide
		newQuestion.IsRainbow = question.IsRainbow
		newQuestion.IsArchive = question.IsArchived
		newQuestion.IsPublish = question.IsPublished
		// Add address prefix
		images := strings.Split(question.Images, ";")
		for i, image := range images {
			if len(image) == 0 {
				continue
			}
			if i != len(images)-1 {
				newQuestion.Images += "https://i0.vjoi.cn/" + image + ";"
			} else {
				newQuestion.Images += "https://i0.vjoi.cn/" + image
			}
		}
		V2DB.Create(&newQuestion)
	}
	log.Info("Question migration finished:", len(oldQuestions))
	// LikeRecord Table
	var oldLikeRecords []oldmodels.LikeRecord
	V1DB.Find(&oldLikeRecords)
	for _, likeRecord := range oldLikeRecords {
		var newLikeRecord database.LikeRecord
		newLikeRecord.ID = likeRecord.ID
		newLikeRecord.CreatedAt = likeRecord.CreatedAt
		newLikeRecord.UpdatedAt = likeRecord.UpdatedAt
		newLikeRecord.IP = likeRecord.IP
		newLikeRecord.QuestionID = likeRecord.QuestionID
		V2DB.Create(&newLikeRecord)
	}
	log.Info("LikeRecord migration finished:", len(oldLikeRecords))
}
