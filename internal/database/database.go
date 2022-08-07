package database

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func Init() {
	var err error
	switch viper.GetString("db_type") {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(viper.GetString("sqlite")), &gorm.Config{})
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", viper.GetString("mysql.user"), viper.GetString("mysql.pass"), viper.GetString("mysql.host"), viper.GetInt("mysql.port"), viper.GetString("mysql.name"))
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
	if err != nil {
		log.Fatal(err)
	}
	migrateAall()
}

func migrateAall() {
	db.AutoMigrate(&Tag{}, &Question{}, &LikeRecord{}, &Admin{}, &Config{})
}
