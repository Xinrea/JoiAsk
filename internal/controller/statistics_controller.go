package controller

import (
	"joiask-backend/internal/database"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type StatisticsController struct{}

type StatisticsResponse struct {
	TotalQuestions     int64     `json:"total_questions"`
	TotalTags          int64     `json:"total_tags"`
	TotalUsers         int64     `json:"total_users"`
	TotalImages        int64     `json:"total_images"`
	RainbowQuestions   int64     `json:"rainbow_questions"`
	ArchivedQuestions  int64     `json:"archived_questions"`
	PublishedQuestions int64     `json:"published_questions"`
	TagStats           []TagStat `json:"tag_stats"`
}

type TagStat struct {
	Tag   database.Tag `json:"tag"`
	Count int64        `json:"count"`
}

func (*StatisticsController) Get(c *gin.Context) {
	var stats StatisticsResponse

	// Count total questions
	if err := database.DB.Model(&database.Question{}).Count(&stats.TotalQuestions).Error; err != nil {
		log.Errorf("failed to count questions: %v", err)
		Fail(c, 500, "内部错误")
		return
	}

	// Count total tags
	if err := database.DB.Model(&database.Tag{}).Count(&stats.TotalTags).Error; err != nil {
		log.Errorf("failed to count tags: %v", err)
		Fail(c, 500, "内部错误")
		return
	}

	// Count total users
	if err := database.DB.Model(&database.Admin{}).Count(&stats.TotalUsers).Error; err != nil {
		log.Errorf("failed to count users: %v", err)
		Fail(c, 500, "内部错误")
		return
	}

	// Sum total images
	var totalImages struct {
		Total int64
	}
	if err := database.DB.Model(&database.Question{}).Select("COALESCE(SUM(images_num), 0) as total").Scan(&totalImages).Error; err != nil {
		log.Errorf("failed to sum images: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	stats.TotalImages = totalImages.Total

	// Count rainbow questions
	if err := database.DB.Model(&database.Question{}).Where("is_rainbow = ?", true).Count(&stats.RainbowQuestions).Error; err != nil {
		log.Errorf("failed to count rainbow questions: %v", err)
		Fail(c, 500, "内部错误")
		return
	}

	// Count archived questions
	if err := database.DB.Model(&database.Question{}).Where("is_archive = ?", true).Count(&stats.ArchivedQuestions).Error; err != nil {
		log.Errorf("failed to count archived questions: %v", err)
		Fail(c, 500, "内部错误")
		return
	}

	// Count published questions
	if err := database.DB.Model(&database.Question{}).Where("is_publish = ?", true).Count(&stats.PublishedQuestions).Error; err != nil {
		log.Errorf("failed to count published questions: %v", err)
		Fail(c, 500, "内部错误")
		return
	}

	// Get tag statistics
	var tags []database.Tag
	if err := database.DB.Find(&tags).Error; err != nil {
		log.Errorf("failed to get tags: %v", err)
		Fail(c, 500, "内部错误")
		return
	}

	stats.TagStats = make([]TagStat, 0, len(tags))
	for _, tag := range tags {
		var count int64
		if err := database.DB.Model(&database.Question{}).Where("tag_id = ?", tag.ID).Count(&count).Error; err != nil {
			log.Errorf("failed to count questions for tag %d: %v", tag.ID, err)
			continue
		}
		stats.TagStats = append(stats.TagStats, TagStat{
			Tag:   tag,
			Count: count,
		})
	}

	Success(c, stats)
}
