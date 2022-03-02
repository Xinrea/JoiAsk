package database

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func Init() {
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", viper.GetString("db.user"), viper.GetString("db.pass"), viper.GetString("db.host"), viper.GetInt("db.port"), viper.GetString("db.name"))
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	migrateAall()
}

func migrateAall() {
	db.AutoMigrate(&Tag{}, &Question{}, &LikeRecord{}, &Admin{}, &Config{})
}
