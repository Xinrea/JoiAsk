package main

import (
	"database/sql"
	"flag"
	"joiask-backend/internal/database"
	"joiask-backend/internal/router"
	"joiask-backend/internal/storage"
	"strings"
	"time"

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
	if *migrate {
		migrateOldData()
	}
	log.Info("Starting server")
	router.Run()
}

func migrateOldData() {
	old, err := sql.Open("sqlite3", "./config/db.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer old.Close()
	// Query all questions
	rows, err := old.Query("SELECT * FROM ask_question")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var (
			id       int
			content  string
			publish  bool
			archive  bool
			thumbsup int
			rainbow  bool
			tag      string
			fold     bool
			filelist string
			date     time.Time
		)
		if err := rows.Scan(&id, &content, &publish, &archive, &thumbsup, &rainbow, &tag, &fold, &filelist, &date); err != nil {
			log.Fatal(err)
		}
		tid, err := database.AddNewTag(strings.Trim(tag, "#"))
		if err != nil {
			log.Fatal(err)
		}
		var q database.Question
		q.Content = content
		q.IsPublished = publish
		q.IsArchived = archive
		q.Likes = thumbsup
		q.IsRainbow = rainbow
		q.TagID = int(tid)
		q.IsHide = fold
		q.Images = filelist
		if len(filelist) > 0 {
			q.ImagesNum = len(strings.Split(filelist, ";"))
		} else {
			q.ImagesNum = 0
		}
		q.CreatedAt = date.Add(-8 * time.Hour)
		if err := database.AddNewQuestion(q); err != nil {
			log.Fatal(err)
		}
	}
}
