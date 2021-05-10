package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	RFC3339Short      = "2006-01-02T15:04:05"
	StartEndTimeParse = "2006-01-02T15:04:05.0000000"
)

type Calendar struct {
	ctx    context.Context
	conf   *oauth2.Config
	client *http.Client

	displayName string
	userName    string
	userMail    string
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

func parseTeamsLink(body string, onlineMeeting interface{}) string {
	if onlineMeeting != nil {
		return onlineMeeting.(map[string]interface{})["joinUrl"].(string)
	}

	re := regexp.MustCompile(`(http|ftp|https)://([\w_-]+(?:(?:\.[\w_-]+)+))([\w.,@?^=%&:/~+#-]*[\w@?^=%&/~+#-])?`)
	links := re.FindAllString(body, -1)

	for _, v := range links {
		if strings.Contains(v, "teams.microsoft.com") {
			return v
		}
	}

	return ""
}

func (c *Calendar) getURL() string {
	return c.conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
}

func (c *Calendar) getRemoteData(url string) ([]byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
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

	body, err := c.getRemoteData("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &user); err != nil {
		return "", err
	}

	c.displayName = user["displayName"].(string)
	c.userMail = user["userPrincipalName"].(string)
	userName := c.userMail[0:strings.Index(c.userMail, "@")]
	c.userName = userName

	for k, v := range loggedUsers {
		if v.userName == userName && k != cookieToken {
			cookieToken = k
			loggedUsers[k] = c
			break
		}
	}

	// TODO Should optimize this and avoid O(2n)
	for k, v := range cachedUsers {
		if userName == k {
			delete(loggedUsers, cookieToken)
			cookieToken = v
			loggedUsers[v] = c
			delete(cachedUsers, k)
			break
		}
	}

	return cookieToken, nil
}

func (c *Calendar) getCalendar() (string, error) {
	var start, end time.Time
	var calData map[string]interface{}

	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch t.Weekday() {
	case time.Saturday:
		start = t.Add(time.Hour * 24 * 2)
	case time.Sunday:
		start = t.Add(time.Hour * 24)
	case time.Monday:
		start = t
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

	url := "https://graph.microsoft.com/v1.0/me/calendarview?startdatetime=" + start.Format(RFC3339Short) + "&enddatetime=" + end.Format(RFC3339Short) + "&top=10&skip=0"
	body, err := c.getRemoteData(url)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &calData); err != nil {
		return "", err
	}

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)
	cal.SetCalscale("GREGORIAN")
	cal.SetName(c.userName)
	cal.SetXWRCalName(c.userName)
	cal.SetDescription("Calendar for user " + c.userName)
	cal.SetXWRCalDesc("Calendar for user " + c.userName)
	cal.SetXWRTimezone("UTC")

	for {
		values := calData["value"].([]interface{})
		for _, v := range values {
			data := v.(map[string]interface{})

			event := cal.AddEvent(data["id"].(string))
			event.SetDtStampTime(time.Now())

			t, _ := time.Parse(time.RFC3339, data["createdDateTime"].(string))
			event.SetCreatedTime(t)

			t, _ = time.Parse(time.RFC3339, data["lastModifiedDateTime"].(string))
			event.SetModifiedAt(t)

			t, _ = time.Parse(StartEndTimeParse, data["start"].(map[string]interface{})["dateTime"].(string))
			event.SetStartAt(t)

			t, _ = time.Parse(StartEndTimeParse, data["end"].(map[string]interface{})["dateTime"].(string))
			event.SetEndAt(t)

			event.SetSummary(data["subject"].(string))
			event.SetLocation(data["location"].(map[string]interface{})["displayName"].(string))

			description := parseTeamsLink(data["body"].(map[string]interface{})["content"].(string), data["onlineMeeting"])
			event.SetDescription(description)

			event.SetURL(data["webLink"].(string))

			organizer := data["organizer"].(map[string]interface{})
			organizerMail := organizer["emailAddress"].(map[string]interface{})["address"].(string)
			organizerName := organizer["emailAddress"].(map[string]interface{})["name"].(string)
			event.SetOrganizer(organizerMail, ics.WithCN(organizerName))

			attendees := data["attendees"].([]interface{})
			for _, att := range attendees {
				var props []ics.PropertyParameter

				castAtt := att.(map[string]interface{})

				typ := castAtt["type"].(string)
				resp := castAtt["status"].(map[string]interface{})["response"].(string)
				name := castAtt["emailAddress"].(map[string]interface{})["name"].(string)

				if typ == "required" {
					props = append(props, ics.ParticipationRoleReqParticipant)
				} else {
					props = append(props, ics.ParticipationRoleOptParticipant)
				}

				switch resp {
				case "accepted":
					props = append(props, ics.ParticipationStatusAccepted)
				case "tentative":
					props = append(props, ics.ParticipationStatusTentative)
				case "declined":
					props = append(props, ics.ParticipationStatusDeclined)
				default:
					props = append(props, ics.ParticipationStatusNeedsAction)
				}

				props = append(props, ics.WithRSVP(true))
				event.AddAttendee(name, props...)
			}
		}

		if nextPage, ok := calData["@odata.nextLink"].(string); ok {
			calData = make(map[string]interface{})
			body, err := c.getRemoteData(nextPage)
			if err != nil {
				return "", err
			}

			if err := json.Unmarshal(body, &calData); err != nil {
				return "", err
			}
		} else {
			break
		}
	}

	return string(cal.Serialize()), nil
}
