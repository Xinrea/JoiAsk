package controller

import (
	"joiask-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

type TagController struct{}

type TagRequest struct {
	TagName     string `json:"tag_name"`
	Description string `json:"description"`
}

// Get all tags
func (t *TagController) Get(c *gin.Context) {
	var tags []database.Tag
	database.DB.Find(&tags)
	Success(c, lo.Map(tags, func(t database.Tag, _ int) interface{} { return t.Json() }))
}

// Put modify tag
func (t *TagController) Put(c *gin.Context) {
	var tag database.Tag
	database.DB.First(&tag, c.Param("id"))
	if tag.ID == 0 {
		Fail(c, 404, "tag not found")
		return
	}
	var tagRequest TagRequest
	if err := c.ShouldBindJSON(&tagRequest); err != nil {
		Fail(c, 400, "invalid request")
		return
	}
	tag.TagName = tagRequest.TagName
	tag.Description = tagRequest.Description
	if err := database.DB.Save(&tag).Error; err != nil {
		log.Errorf("failed to save tag: %v", err)
		Fail(c, 500, "internal server error")
		return
	}
	Success(c, tag)
}

// Post create tag
func (t *TagController) Post(c *gin.Context) {
	var tagRequest TagRequest
	if err := c.ShouldBindJSON(&tagRequest); err != nil {
		Fail(c, 400, "invalid request")
		return
	}
	var tag database.Tag
	tag.TagName = tagRequest.TagName
	tag.Description = tagRequest.Description
	if err := database.DB.Create(&tag).Error; err != nil {
		log.Error("failed to create tag: ", err)
		Fail(c, 401, "创建话题失败")
		return
	}
	Success(c, tag)
}

func (t *TagController) Delete(c *gin.Context) {
	var tag database.Tag
	database.DB.First(&tag, c.Param("id"))
	if tag.ID == 0 {
		Fail(c, 404, "话题不存在")
		return
	}
	if tag.ID == 1 {
		Fail(c, 403, "不能删除默认话题")
		return
	}
	var questionCount int64
	database.DB.Model(&database.Question{}).Where("tag_id = ?", tag.ID).Count(&questionCount)
	if questionCount > 0 {
		Fail(c, 400, "话题仍在使用中")
		return
	}
	if err := database.DB.Delete(&tag).Error; err != nil {
		log.Error("failed to delete tag: ", err.Error())
		Fail(c, 500, "internal server error")
		return
	}
	Success(c, nil)
}
