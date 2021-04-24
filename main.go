package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	RFC3339Short = "2006-01-02T15:04:05"
)

func main() {
	// TODO Store all information on a DB (PostgreSQL because RPi)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     viper.GetString("client_id"),
		ClientSecret: viper.GetString("secret"),
		Scopes:       []string{"offline_access", "user.read", "calendars.read"},
		RedirectURL:  viper.GetString("redirect_url"),
		Endpoint:     microsoft.AzureADEndpoint(viper.GetString("tenant")),
	}

	codeChan := make(chan string)
	go web(conf.AuthCodeURL("state", oauth2.AccessTypeOffline), codeChan)

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	code := <-codeChan

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}

	t := time.Now()
	today := t.Format(RFC3339Short)
	nextWeek := t.Add(time.Hour * 24 * 7).Format(RFC3339Short)

	client := conf.Client(ctx, tok)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me/calendarview?startdatetime=" + today + "&enddatetime=" + nextWeek)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(body))
}
