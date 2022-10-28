package database

import (
	"fmt"
	"joiask-backend/pkg/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

const DefaultTagName = "提问箱"

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
	err := DB.AutoMigrate(&Question{}, &LikeRecord{}, &Admin{}, &Config{}, &Tag{})
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
	if DB.First(&Config{}).RowsAffected == 0 {
		log.Info("Initializing default config.")
		if err := DB.Create(&Config{Announcement: "提问内容将在审核后公开"}).Error; err != nil {
			log.Fatal("Failed to initialize default config.", err)
		}
	}
	// Initialize default tag.
	if DB.First(&Tag{}).RowsAffected == 0 {
		log.Info("Initializing default tag.")
		if err := DB.Create(&Tag{TagName: DefaultTagName, Description: "默认话题"}).Error; err != nil {
			log.Fatal("Failed to initialize default tag.", err)
		}
	}
}
