package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var loggedUsers map[string]*Calendar
var cachedUsers map[string]string

func main() {
	// TODO Add logout to clear user
	// TODO Improve pagination (from 10 to 30)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(-1)
	}

	loggedUsers = make(map[string]*Calendar)
	rand.Seed(time.Now().UnixNano())

	mysqlConfs := viper.GetStringMapString("mysql")

	err := initCache(&DBConfs{
		user:     mysqlConfs["user"],
		password: mysqlConfs["password"],
		host:     mysqlConfs["host"],
		schema:   mysqlConfs["schema"],
	})

	if err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(-1)
	}

	cachedUsers, err = cachedData.loadUserTokens()
	if err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(-1)
	}

	web()
}
