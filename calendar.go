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

func (c *Calendar) handleToken(code string, cookieToken string) (string, error) {
	var user map[string]interface{}

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.

	tok, err := c.conf.Exchange(c.ctx, code)
	if err != nil {
		return "", err
	}

	c.client = c.conf.Client(c.ctx, tok)

	resp, err := c.client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &user); err != nil {
		return "", err
	}

	userName := user["userPrincipalName"].(string)
	userName = userName[0:strings.Index(userName, "@")]
	c.userName = userName

	for k, v := range loggedUsers {
		if v.userName == userName && k != cookieToken {
			cookieToken = k
			loggedUsers[k] = c
			break
		}
	}

	return cookieToken, nil
}

func (c *Calendar) getCalendar() string {
	var start, end time.Time

	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch t.Weekday() {
	case time.Saturday:
		start = t.Add(time.Hour * 24 * 2)
	case time.Sunday:
		start = t.Add(time.Hour * 24)
	default:
		start = t
		for {
			start = start.Add(time.Hour * 24 * -1)
			if start.Weekday() == time.Monday {
				break
			}
		}
	}

	end = start.Add(time.Hour * 24 * 5)

	resp, err := c.client.Get("https://graph.microsoft.com/v1.0/me/calendarview?startdatetime=" + start.Format(RFC3339Short) + "&enddatetime=" + end.Format(RFC3339Short))
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
