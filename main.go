package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var loggedUsers map[string]*Calendar
var cachedUsers map[string]string

var BuildDate string

func main() {
	// TODO Add logout to clear user

	var err error
	var sqlConfs map[string]string

	fmt.Printf("O365 to iCal build from %s\n", BuildDate)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(-1)
	}

	loggedUsers = make(map[string]*Calendar)
	rand.Seed(time.Now().UnixNano())

	sqlConfs = viper.GetStringMapString("psql")

	err = initCache(&DBConfs{
		user:     sqlConfs["user"],
		password: sqlConfs["password"],
		host:     sqlConfs["host"],
		schema:   sqlConfs["schema"],
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

	go refreshCache()

	web()
}
