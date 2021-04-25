package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	RFC3339Short = "2006-01-02T15:04:05"
)

type Calendar struct {
	ctx    context.Context
	conf   *oauth2.Config
	client *http.Client

	userName    string
	lastUpdated time.Time
}

func newCalendarHandler() *Calendar {
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     viper.GetString("client_id"),
		ClientSecret: viper.GetString("secret"),
		Scopes:       []string{"offline_access", "user.read", "calendars.read"},
		RedirectURL:  viper.GetString("redirect_url"),
		Endpoint:     microsoft.AzureADEndpoint(viper.GetString("tenant")),
	}

	return &Calendar{
		ctx:         ctx,
		conf:        conf,
		lastUpdated: time.Now(),
	}
}

func (c *Calendar) getURL() string {
	return c.conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
}

func (c *Calendar) handleToken(code string) error {
	var user map[string]interface{}

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.

	tok, err := c.conf.Exchange(c.ctx, code)
	if err != nil {
		return err
	}

	c.client = c.conf.Client(c.ctx, tok)

	resp, err := c.client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &user); err != nil {
		return err
	}

	userName := user["userPrincipalName"].(string)
	userName = userName[0:strings.Index(userName, "@")]
	c.userName = userName

	return nil
}

func (c *Calendar) getCalendar() string {
	t := time.Now()
	today := t.Format(RFC3339Short)
	nextWeek := t.Add(time.Hour * 24 * 7).Format(RFC3339Short)

	resp, err := c.client.Get("https://graph.microsoft.com/v1.0/me/calendarview?startdatetime=" + today + "&enddatetime=" + nextWeek)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(body)
}
