package controller

import (
	"errors"
	neturl "net/url"
	"strings"

	"joiask-backend/internal/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	maxEmojiTagLength = 255
	maxEmojiURLLength = 1024
)

type EmojiController struct{}

type EmojiRequest struct {
	Tag string `json:"tag"`
	URL string `json:"url"`
}

func validateEmojiRequest(request EmojiRequest) (string, string, bool) {
	tag, url := strings.TrimSpace(request.Tag), strings.TrimSpace(request.URL)
	parsedURL, urlErr := neturl.Parse(url)
	validURL := urlErr == nil &&
		((strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "//")) ||
			((parsedURL.Scheme == "http" || parsedURL.Scheme == "https") &&
				parsedURL.Host != "" && parsedURL.User == nil))
	return tag, url, tag != "" && url != "" &&
		len([]rune(tag)) <= maxEmojiTagLength && len([]rune(url)) <= maxEmojiURLLength &&
		validURL
}

func (*EmojiController) Get(c *gin.Context) {
	emojis := []database.Emoji{}
	if err := database.DB.Order("id asc").Find(&emojis).Error; err != nil {
		Fail(c, 500, "internal server error")
		return
	}
	Success(c, emojis)
}

func (*EmojiController) Post(c *gin.Context) {
	var request EmojiRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "invalid request")
		return
	}
	tag, url, ok := validateEmojiRequest(request)
	if !ok {
		Fail(c, 400, "tag 和 url 不能为空，且 url 必须是站内路径或 HTTP(S) 地址")
		return
	}
	var duplicate database.Emoji
	if database.DB.Where("tag = ?", tag).First(&duplicate).Error == nil {
		Fail(c, 400, "tag already exists")
		return
	}
	emoji := database.Emoji{Tag: tag, URL: url}
	if err := database.DB.Create(&emoji).Error; err != nil {
		Fail(c, 400, "tag already exists")
		return
	}
	Success(c, emoji)
}

func (*EmojiController) Put(c *gin.Context) {
	var emoji database.Emoji
	if err := database.DB.First(&emoji, c.Param("id")).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		Fail(c, 404, "emoji not found")
		return
	} else if err != nil {
		Fail(c, 500, "internal server error")
		return
	}
	var request EmojiRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "invalid request")
		return
	}
	tag, url, ok := validateEmojiRequest(request)
	if !ok {
		Fail(c, 400, "tag 和 url 不能为空，且 url 必须是站内路径或 HTTP(S) 地址")
		return
	}
	var duplicate database.Emoji
	if database.DB.Where("tag = ? AND id <> ?", tag, emoji.ID).First(&duplicate).Error == nil {
		Fail(c, 400, "tag already exists")
		return
	}
	emoji.Tag, emoji.URL = tag, url
	if err := database.DB.Save(&emoji).Error; err != nil {
		Fail(c, 400, "tag already exists")
		return
	}
	Success(c, emoji)
}

func (*EmojiController) Delete(c *gin.Context) {
	var emoji database.Emoji
	if err := database.DB.First(&emoji, c.Param("id")).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		Fail(c, 404, "emoji not found")
		return
	} else if err != nil {
		Fail(c, 500, "internal server error")
		return
	}
	if err := database.DB.Delete(&emoji).Error; err != nil {
		Fail(c, 500, "internal server error")
		return
	}
	Success(c, nil)
}
