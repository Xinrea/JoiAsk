package main

import (
	"joiask-backend/internal/database"
	"joiask-backend/internal/router"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigFile("./config/config.json")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Config loaded")
	database.Init()
	log.Info("Database initialized")
	log.Info("Starting server")
	router.Run()
}
