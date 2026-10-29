package main

import (
	"errors"
	"fmt"
	"io"
	"joiask-backend/internal/database"
	"joiask-backend/internal/router"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	if len(args) != 0 && (len(args) != 3 || args[0] != "admin" ||
		(args[1] != "create" && args[1] != "reset-password" && args[1] != "promote") || strings.TrimSpace(args[2]) == "") {
		return errors.New("usage: jask [admin create <username> | admin reset-password <username> | admin promote <username>]; stop all backend instances first; create/reset password is read from stdin")
	}
	configFile := os.Getenv("JOIASK_CONFIG")
	if configFile == "" {
		configFile = "./config/config.json"
	}
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		return errors.New("unable to read configuration; check JOIASK_CONFIG or ./config/config.json")
	}
	if len(args) != 0 {
		var password string
		if args[1] != "promote" {
			password, err = readPassword(input)
			if err != nil {
				return err
			}
		}
		db, err := database.OpenAdminDatabase()
		if err != nil {
			return err
		}
		sqlDB, err := db.DB()
		if err != nil {
			return errors.New("unable to access administrator database")
		}
		defer sqlDB.Close()
		if args[1] == "promote" {
			err = database.PromoteAdmin(db, args[2])
		} else {
			err = database.SetAdminPassword(db, args[2], password, args[1] == "reset-password")
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "Administrator account updated.")
		return err
	}
	log.Info("Config loaded")
	store, err := router.NewSessionStore(viper.GetString("security.session_secret"), viper.GetBool("security.secure_cookie"))
	if err != nil {
		return err
	}
	database.Init()
	log.Info("Database initialized")
	log.Info("Starting server")
	router.Run(store)
	return nil
}

func readPassword(input io.Reader) (string, error) {
	// Allow one optional LF/CRLF, preserve all other password bytes, and bound
	// input before hashing. The extra byte detects oversized input.
	data, err := io.ReadAll(io.LimitReader(input, 4099))
	if err != nil {
		return "", errors.New("unable to read password from stdin")
	}
	password := strings.TrimSuffix(string(data), "\n")
	if len(password) != len(data) {
		password = strings.TrimSuffix(password, "\r")
	}
	if len(password) == 0 || len(password) > 4096 || strings.ContainsAny(password, "\r\n") {
		return "", errors.New("stdin must contain one non-empty password of at most 4096 bytes, optionally followed by a newline")
	}
	return password, nil
}
