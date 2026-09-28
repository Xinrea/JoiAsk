package controller

import (
	"bytes"
	"io"
	"joiask-backend/internal/database"
	"joiask-backend/internal/storage"
	"joiask-backend/pkg/util"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ConfigController struct{}

type ConfigRequest struct {
	SiteName                  *string `json:"site_name"`
	SiteDescription           *string `json:"site_description"`
	LogoURL                   *string `json:"logo_url"`
	FaviconURL                *string `json:"favicon_url"`
	CustomCSS                 *string `json:"custom_css"`
	Announcement              *string `json:"announcement"`
	RequireVerifiedUserToPost *bool   `json:"require_verified_user_to_post"`
}

type SettingsRequest struct {
	DeepSeekAPIKey            string `json:"deepseek_api_key"`
	SpamPrompt                string `json:"spam_prompt"`
	RequireVerifiedUserToPost bool   `json:"require_verified_user_to_post"`
}

type AssetResponse struct {
	URL string `json:"url"`
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
	if request.SiteName != nil {
		config.SiteName = strings.TrimSpace(*request.SiteName)
		if config.SiteName == "" {
			config.SiteName = database.DefaultSiteName
		}
	}
	if request.SiteDescription != nil {
		config.SiteDescription = strings.TrimSpace(*request.SiteDescription)
		if config.SiteDescription == "" {
			config.SiteDescription = database.DefaultSiteDescription
		}
	}
	if request.LogoURL != nil {
		config.LogoURL = strings.TrimSpace(*request.LogoURL)
		if config.LogoURL == "" {
			config.LogoURL = database.DefaultLogoURL
		}
	}
	if request.FaviconURL != nil {
		config.FaviconURL = strings.TrimSpace(*request.FaviconURL)
		if config.FaviconURL == "" {
			config.FaviconURL = database.DefaultFaviconURL
		}
	}
	if request.CustomCSS != nil {
		if strings.Contains(strings.ToLower(*request.CustomCSS), "</style") {
			Fail(c, 400, "自定义 CSS 不能包含 </style>")
			return
		}
		if len(*request.CustomCSS) > 128<<10 {
			Fail(c, 400, "自定义 CSS 不能超过 128 KiB")
			return
		}
		config.CustomCSS = *request.CustomCSS
	}
	if request.Announcement != nil {
		config.Announcement = *request.Announcement
	}
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

func (*ConfigController) UploadAsset(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, 400, "请选择图片文件")
		return
	}
	if file.Size <= 0 || file.Size > 5<<20 {
		Fail(c, 400, "图片大小不能超过 5 MB")
		return
	}

	opened, err := file.Open()
	if err != nil {
		Fail(c, 400, "图片读取失败")
		return
	}
	defer opened.Close()

	content, err := io.ReadAll(io.LimitReader(opened, 5<<20+1))
	if err != nil || len(content) == 0 || len(content) > 5<<20 {
		Fail(c, 400, "图片读取失败")
		return
	}
	contentType := http.DetectContentType(content)
	extension := assetExtension(content, contentType)
	if extension == "" {
		Fail(c, 400, "仅支持 PNG、JPEG、GIF、WebP 或 SVG 图片")
		return
	}

	filename := "site-asset-" + util.Md5v(string(content)) + extension
	url, err := storage.Get().Upload(filename, bytes.NewReader(content))
	if err != nil {
		log.Errorf("failed to upload site asset: %v", err)
		Fail(c, 500, "图片上传失败")
		return
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	Success(c, AssetResponse{URL: url})
}

func assetExtension(content []byte, contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	}
	trimmed := strings.TrimSpace(string(content))
	if strings.HasPrefix(trimmed, "<svg") || strings.HasPrefix(trimmed, "<?xml") && strings.Contains(trimmed, "<svg") {
		return ".svg"
	}
	return ""
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
