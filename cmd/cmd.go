package main

import (
	"flag"
	"joiask-backend/internal/database"
	"joiask-backend/internal/router"
	"joiask-backend/internal/storage"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var migrate = flag.Bool("migrate", false, "migrate")

func main() {
	flag.Parse()
	viper.SetConfigFile("./config/config.json")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Config loaded")
	database.Init()
	log.Info("Database initialized")
	storage.Init()
	log.Info("Storage initialized")
	log.Info("Starting server")
	router.Run()
}
