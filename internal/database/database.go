package database

import (
	"fmt"
	"joiask-backend/internal/deepseek"
	"joiask-backend/pkg/util"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var DB *gorm.DB

const DefaultTagName = "提问箱"
const DefaultSiteName = "JoiAsk 提问箱"
const DefaultSiteDescription = "JoiAsk 提问箱"
const DefaultLogoURL = "/favicon.png"
const DefaultFaviconURL = "/favicon.png"

var DefaultEmojis = []Emoji{
	{Tag: "[轴伊Joi收藏集动态表情包_跑了]", URL: "/joi-emojis/paole.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_鞠躬]", URL: "/joi-emojis/jugong.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_摇你]", URL: "/joi-emojis/yaoni.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_愤怒]", URL: "/joi-emojis/fennu.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_猴]", URL: "/joi-emojis/hou.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_NO]", URL: "/joi-emojis/no.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_贴贴]", URL: "/joi-emojis/tietie.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_呆]", URL: "/joi-emojis/dai.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_唔唔]", URL: "/joi-emojis/wuwu.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_啊这]", URL: "/joi-emojis/azhe.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_失落]", URL: "/joi-emojis/shiluo.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_神气]", URL: "/joi-emojis/shenqi.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_怎么这样]", URL: "/joi-emojis/zenmezhyang.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_睡觉]", URL: "/joi-emojis/shuijiao.webp"},
	{Tag: "[轴伊Joi收藏集动态表情包_爆]", URL: "/joi-emojis/bao.webp"},
}

// Init opens connection and try to initialize the database.
func Init() {
	var err error
	switch viper.GetString("db_type") {
	case "sqlite":
		log.Info("Using sqlite database.")
		DB, err = gorm.Open(sqlite.Open(viper.GetString("sqlite")), &gorm.Config{})
	case "mysql":
		log.Info("Using mysql database.")
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True", viper.GetString("mysql.user"), viper.GetString("mysql.pass"), viper.GetString("mysql.host"), viper.GetInt("mysql.port"), viper.GetString("mysql.name"))
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
	if err != nil {
		log.Fatal(err)
	}
	initializeDB()
}

// initializeDB initializes the database, create tables and default records.
func initializeDB() {
	err := DB.AutoMigrate(
		&User{},
		&BilibiliVerificationAccount{},
		&BilibiliVerificationRequest{},
		&Question{},
		&LikeRecord{},
		&Admin{},
		&Config{},
		&Tag{},
		&Emoji{},
	)
	if err != nil {
		log.Fatal(err)
	}
	// Initialize default admin account.
	if DB.Where("username = ?", "admin").First(&Admin{}).RowsAffected == 0 {
		log.Info("Initializing default admin account.")
		if err := DB.Create(&Admin{Username: "admin", Password: util.Md5v("admin")}).Error; err != nil {
			log.Fatal("Failed to initialize default admin account.", err)
		}
	}
	// Initialize default config.
	var config Config
	if DB.First(&config).RowsAffected == 0 {
		log.Info("Initializing default config.")
		if err := DB.Create(&Config{
			SiteName:        DefaultSiteName,
			SiteDescription: DefaultSiteDescription,
			LogoURL:         DefaultLogoURL,
			FaviconURL:      DefaultFaviconURL,
			Announcement:    "提问内容将在审核后公开",
			SpamPrompt:      deepseek.DefaultSpamPrompt,
		}).Error; err != nil {
			log.Fatal("Failed to initialize default config.", err)
		}
	} else {
		updates := map[string]interface{}{}
		if strings.TrimSpace(config.SiteName) == "" {
			updates["site_name"] = DefaultSiteName
		}
		if strings.TrimSpace(config.SiteDescription) == "" {
			updates["site_description"] = DefaultSiteDescription
		}
		if strings.TrimSpace(config.LogoURL) == "" {
			updates["logo_url"] = DefaultLogoURL
		}
		if strings.TrimSpace(config.FaviconURL) == "" {
			updates["favicon_url"] = DefaultFaviconURL
		}
		if strings.TrimSpace(config.SpamPrompt) == "" {
			updates["spam_prompt"] = deepseek.DefaultSpamPrompt
		}
		if len(updates) > 0 {
			if err := DB.Model(&config).Updates(updates).Error; err != nil {
				log.Fatal("Failed to initialize config defaults.", err)
			}
		}
	}
	// Initialize default tag.
	if DB.First(&Tag{}).RowsAffected == 0 {
		log.Info("Initializing default tag.")
		if err := DB.Create(&Tag{TagName: DefaultTagName, Description: "默认话题"}).Error; err != nil {
			log.Fatal("Failed to initialize default tag.", err)
		}
	}
	if err := initializeEmojis(DB); err != nil {
		log.Fatal("Failed to initialize default emojis.", err)
	}
}

// Seed once, atomically with the marker, so edits/deletions (even deleting all
// entries) survive restarts and an interrupted initialization can be retried.
func initializeEmojis(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var config Config
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&config).Error; err != nil {
			return err
		}
		if config.EmojisInitialized {
			return nil
		}
		for _, emoji := range DefaultEmojis {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&emoji).Error; err != nil {
				return err
			}
		}
		return tx.Model(&config).Update("emojis_initialized", true).Error
	})
}
