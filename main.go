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
	var hasDBConn bool
	var sqlConfs map[string]string
	var sqlDriver string

	fmt.Printf("O365 to iCal build from %s\n", BuildDate)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(-1)
	}

	loggedUsers = make(map[string]*Calendar)
	rand.Seed(time.Now().UnixNano())

	if viper.IsSet("mysql") {
		sqlDriver = "mysql"
		sqlConfs = viper.GetStringMapString("mysql")
		hasDBConn = true
	}

	if viper.IsSet("psql") {
		sqlDriver = "postgres"
		sqlConfs = viper.GetStringMapString("psql")
		hasDBConn = true
	}

	if hasDBConn {
		err = initCache(&DBConfs{
			user:     sqlConfs["user"],
			password: sqlConfs["password"],
			host:     sqlConfs["host"],
			schema:   sqlConfs["schema"],
		}, sqlDriver)

		if err != nil {
			log.Fatal().Err(err).Send()
			os.Exit(-1)
		}
	} else {
		log.Fatal().
			Msg("No database connection found in configuration file")
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
