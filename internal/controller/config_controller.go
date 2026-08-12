package controller

import (
	"joiask-backend/internal/database"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ConfigController struct{}

type ConfigRequest struct {
	Announcement string `json:"announcement"`
}

type SettingsRequest struct {
	DeepSeekAPIKey string `json:"deepseek_api_key"`
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
	Success(c, SettingsRequest{DeepSeekAPIKey: config.DeepSeekAPIKey})
}

func (*ConfigController) PutSettings(c *gin.Context) {
	var request SettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		Fail(c, 400, "请求错误")
		return
	}

	var config database.Config
	if err := database.DB.First(&config).Error; err != nil {
		log.Errorf("failed to get settings: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	config.DeepSeekAPIKey = request.DeepSeekAPIKey
	if err := database.DB.Save(&config).Error; err != nil {
		log.Errorf("failed to save settings: %v", err)
		Fail(c, 500, "内部错误")
		return
	}
	Success(c, SettingsRequest{DeepSeekAPIKey: config.DeepSeekAPIKey})
}
