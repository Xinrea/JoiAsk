package controller

import (
	"bytes"
	"joiask-backend/internal/database"
	"joiask-backend/internal/imagefile"
	"joiask-backend/internal/mailer"
	"joiask-backend/internal/storage"
	"joiask-backend/pkg/util"
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
	Email                     *mailer.Settings `json:"email,omitempty"`
	DeepSeekAPIKey            string           `json:"deepseek_api_key"`
	SpamPrompt                string           `json:"spam_prompt"`
	RequireVerifiedUserToPost bool             `json:"require_verified_user_to_post"`
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
	if file.Size <= 0 || file.Size > imagefile.MaxSize {
		Fail(c, 400, "图片大小不能超过 5 MB")
		return
	}

	opened, err := file.Open()
	if err != nil {
		Fail(c, 400, "图片读取失败")
		return
	}
	defer opened.Close()

	content, err := imagefile.Read(opened)
	if err != nil {
		Fail(c, 400, "图片读取失败")
		return
	}
	extension := assetExtension(content)
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

func assetExtension(content []byte) string {
	if extension := imagefile.RasterExtension(content); extension != "" {
		return extension
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
		Email:                     publicMailSettings(config),
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
	if request.Email != nil {
		mail := *request.Email
		mail.Normalize()
		if mail.Password == "" {
			mail.Password = config.SMTPPassword
		}
		if mail.Enabled {
			if err := mail.Validate(); err != nil {
				Fail(c, 400, err.Error())
				return
			}
		}
		config.EmailEnabled = mail.Enabled
		config.SMTPHost = mail.Host
		config.SMTPPort = mail.Port
		config.SMTPUsername = mail.Username
		config.SMTPPassword = mail.Password
		config.SMTPSecurity = mail.Security
		config.SMTPFromEmail = mail.FromEmail
		config.SMTPFromName = mail.FromName
		config.EmailSiteURL = mail.SiteURL
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
		Email:                     publicMailSettings(config),
		DeepSeekAPIKey:            config.DeepSeekAPIKey,
		SpamPrompt:                config.SpamPrompt,
		RequireVerifiedUserToPost: config.RequireVerifiedUserToPost,
	})
}

func mailSettings(config database.Config) mailer.Settings {
	settings := mailer.Settings{Enabled: config.EmailEnabled, Host: config.SMTPHost, Port: config.SMTPPort, Username: config.SMTPUsername, Password: config.SMTPPassword, Security: config.SMTPSecurity, FromEmail: config.SMTPFromEmail, FromName: config.SMTPFromName, SiteURL: config.EmailSiteURL}
	settings.Normalize()
	return settings
}

func publicMailSettings(config database.Config) *mailer.Settings {
	settings := mailSettings(config)
	settings.PasswordConfigured = settings.Password != ""
	settings.Password = ""
	return &settings
}
