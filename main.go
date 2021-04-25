package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/spf13/viper"
)

var loggedUsers map[string]*Calendar

func main() {
	// TODO Pick current workweek, from Mon to Fri
	// TODO Add logout to clear user

	// TODO LOW Store user token in DB to persist across restarts

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	loggedUsers = make(map[string]*Calendar)
	rand.Seed(time.Now().UnixNano())

	web()
}
