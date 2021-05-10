package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var loggedUsers map[string]*Calendar

func main() {
	// TODO Add logout to clear user
	// TODO LOW Store user token in DB to persist across restarts

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(-1)
	}

	loggedUsers = make(map[string]*Calendar)
	rand.Seed(time.Now().UnixNano())

	mysqlConfs := viper.GetStringMapString("mysql")

	_, err := initCache(&DBConfs{
		user:     mysqlConfs["user"],
		password: mysqlConfs["password"],
		host:     mysqlConfs["host"],
		schema:   mysqlConfs["schema"],
	})

	if err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(-1)
	}

	web()
}
