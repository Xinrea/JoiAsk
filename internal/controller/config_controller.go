package controller

import (
	"joiask-backend/internal/database"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ConfigController struct{}

type ConfigRequest struct {
	Announcement              string `json:"announcement"`
	RequireVerifiedUserToPost *bool  `json:"require_verified_user_to_post"`
}

type SettingsRequest struct {
	DeepSeekAPIKey            string `json:"deepseek_api_key"`
	SpamPrompt                string `json:"spam_prompt"`
	RequireVerifiedUserToPost bool   `json:"require_verified_user_to_post"`
}

func (*ConfigController) Get(c *gin.Context) {
	var config database.Config
	database.DB.First(&config)
	Success(c, config)
}

func (*ConfigController) Put(c *gin.Context) {
	var request ConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求错误")
		return
	}
	var config database.Config
	database.DB.First(&config)
	config.Announcement = request.Announcement
	if request.RequireVerifiedUserToPost != nil {
		config.RequireVerifiedUserToPost = *request.RequireVerifiedUserToPost
	}
	if err := database.DB.Save(&config).Error; err != nil {
		log.Errorf("failed to save config: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, config)
}

func (*ConfigController) GetSettings(c *gin.Context) {
	var config database.Config
	if err := database.DB.First(&config).Error; err != nil {
		log.Errorf("failed to get settings: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, SettingsRequest{
		DeepSeekAPIKey:            config.DeepSeekAPIKey,
		SpamPrompt:                config.SpamPrompt,
		RequireVerifiedUserToPost: config.RequireVerifiedUserToPost,
	})
}

func (*ConfigController) PutSettings(c *gin.Context) {
	var request SettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求错误")
		return
	}
	request.SpamPrompt = strings.TrimSpace(request.SpamPrompt)
	if request.SpamPrompt == "" {
		Fail(c, 400, "低质量提问判定标准不能为空")
		return
	}

	var config database.Config
	if err := database.DB.First(&config).Error; err != nil {
		log.Errorf("failed to get settings: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	config.DeepSeekAPIKey = request.DeepSeekAPIKey
	config.SpamPrompt = request.SpamPrompt
	config.RequireVerifiedUserToPost = request.RequireVerifiedUserToPost
	if err := database.DB.Save(&config).Error; err != nil {
		log.Errorf("failed to save settings: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, SettingsRequest{
		DeepSeekAPIKey:            config.DeepSeekAPIKey,
		SpamPrompt:                config.SpamPrompt,
		RequireVerifiedUserToPost: config.RequireVerifiedUserToPost,
	})
}
