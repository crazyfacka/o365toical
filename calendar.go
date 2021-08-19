package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/rs/zerolog/log"
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
	valid       bool
	lastUpdated time.Time
}

type Attachment struct {
	url      string
	mimeType string
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
		valid:       false,
		lastUpdated: time.Now(),
	}
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

func (c *Calendar) saveURLToFile(url string, attId string, fname string) error {
	baseDir := viper.GetString("attachments_dir") + "/" + attId
	os.MkdirAll(baseDir, os.ModePerm)

	file, err := os.Create(baseDir + "/" + fname)
	if err != nil {
		return err
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	defer file.Close()

	return nil
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

	c.valid = true

	return cookieToken, nil
}

func (c *Calendar) shouldSkip(data map[string]interface{}) bool {
	rspStatus := data["responseStatus"].(map[string]interface{})
	if rspStatus["response"].(string) != "accepted" && rspStatus["response"].(string) != "organizer" && rspStatus["response"].(string) != "none" {
		return true
	}

	if data["isAllDay"].(bool) {
		return true
	}

	if data["showAs"].(string) != "busy" {
		return true
	}

	return false
}

func (c *Calendar) handleAttachments(baseHost, id string, hasAttachments bool) ([]*Attachment, error) {
	if !hasAttachments {
		return nil, nil
	}

	var attData map[string]interface{}
	var attachments []*Attachment

	baseUrl := "https://graph.microsoft.com/v1.0/me/events/" + id + "/attachments"
	body, err := c.getRemoteData(baseUrl)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &attData); err != nil {
		return nil, err
	}

	values := attData["value"].([]interface{})
	for _, v := range values {
		data := v.(map[string]interface{})

		attId := data["id"].(string)
		name := data["name"].(string)

		var contentType string
		if val, ok := data["contentType"]; ok && val != nil {
			contentType = val.(string)
		} else {
			contentType = "application/octet-stream"
		}

		if attCache := cachedData.attachmentExists(attId); attCache != nil {
			attachments = append(attachments, &Attachment{
				url:      "https://" + baseHost + "/attachment/" + attId + "/" + url.PathEscape(attCache[0]),
				mimeType: attCache[1],
			})
			continue
		}

		go func() {
			baseUrl := "https://graph.microsoft.com/v1.0/me/events/" + id + "/attachments/" + attId + "/$value"
			if err := c.saveURLToFile(baseUrl, attId, name); err != nil {
				log.Error().
					Err(err).
					Str("Attachment ID", attId).
					Str("File name", name).
					Msg("Error saving file to disk")
			}
		}()

		if err = cachedData.saveAttachment(attId, name, contentType); err != nil {
			return nil, err
		}

		attachments = append(attachments, &Attachment{
			url:      "https://" + baseHost + "/attachment/" + attId + "/" + url.PathEscape(name),
			mimeType: contentType,
		})
	}

	return attachments, nil
}

func (c *Calendar) handleBasicEventData(cal *ics.Calendar, data map[string]interface{}) *ics.VEvent {
	event := cal.AddEvent(data["id"].(string))
	event.SetDtStampTime(time.Now())

	t, _ := time.Parse(time.RFC3339, data["createdDateTime"].(string))
	event.SetCreatedTime(t)

	t, _ = time.Parse(time.RFC3339, data["lastModifiedDateTime"].(string))
	event.SetModifiedAt(t)

	ts, _ := time.Parse(StartEndTimeParse, data["start"].(map[string]interface{})["dateTime"].(string))
	te, _ := time.Parse(StartEndTimeParse, data["end"].(map[string]interface{})["dateTime"].(string))

	if data["isAllDay"].(bool) {
		event.SetAllDayStartAt(ts)
		event.SetAllDayEndAt(te)
	} else {
		event.SetStartAt(ts)
		event.SetEndAt(te)
	}

	event.SetSummary(data["subject"].(string))
	event.SetLocation(data["location"].(map[string]interface{})["displayName"].(string))

	return event
}

func (c *Calendar) handleDescription(event *ics.VEvent, data map[string]interface{}, atts []*Attachment) {
	link := strings.TrimSpace(parseTeamsLink(data["body"].(map[string]interface{})["content"].(string), data["onlineMeeting"]))
	if link != "" {
		event.SetURL(link)
	}

	var attString strings.Builder
	for i, v := range atts {
		if i > 0 {
			attString.WriteString("\n\n")
		}

		attString.WriteString("Attachment ")
		attString.WriteString("(")
		attString.WriteString(strconv.Itoa(i + 1))
		attString.WriteString("): ")
		attString.WriteString(v.url)
	}

	description, err := html2text(data["body"].(map[string]interface{})["content"].(string))
	if err == nil && description != "" {
		var dscString strings.Builder
		dscString.WriteString(description)

		if attString.Len() > 0 {
			dscString.WriteString("\n\n")
			dscString.WriteString(attString.String())
		}

		if link != "" {
			dscString.WriteString("\n\n")
			dscString.WriteString(link)
		}

		event.SetDescription(dscString.String())
	}
}

func (c *Calendar) handleAttendees(event *ics.VEvent, data map[string]interface{}, google bool) {
	organizer := data["organizer"].(map[string]interface{})
	organizerMail := organizer["emailAddress"].(map[string]interface{})["address"].(string)
	organizerName := organizer["emailAddress"].(map[string]interface{})["name"].(string)
	event.SetOrganizer(organizerMail, ics.WithCN(organizerName))

	event.AddAttendee(organizerMail, ics.ParticipationRoleChair, ics.ParticipationStatusAccepted, ics.WithCN(organizerName))

	// Google can't handle big lists of invitees (>5 I guess), and don't display them either way
	if !google {
		attendees := data["attendees"].([]interface{})
		for _, att := range attendees {
			var props []ics.PropertyParameter

			castAtt := att.(map[string]interface{})

			typ := castAtt["type"].(string)
			resp := castAtt["status"].(map[string]interface{})["response"].(string)
			name := castAtt["emailAddress"].(map[string]interface{})["name"].(string)
			email := castAtt["emailAddress"].(map[string]interface{})["address"].(string)

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

			props = append(props, ics.WithCN(name))
			event.AddAttendee(email, props...)
		}
	}
}

func (c *Calendar) getCalendar(baseHost string, full bool, google bool) (string, error) {
	var calData map[string]interface{}
	var cacheRetrieved bool

	start, end := getStartEndWeekDays()

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
	cal.SetXWRTimezone("UTC")

	for {
		values := calData["value"].([]interface{})
		for _, v := range values {
			data := v.(map[string]interface{})

			if !full && c.shouldSkip(data) {
				continue
			}

			event := c.handleBasicEventData(cal, data)

			// Google only supports attachments that are hosted on Drive
			var atts []*Attachment
			if !google {
				atts, err = c.handleAttachments(baseHost, data["id"].(string), data["hasAttachments"].(bool))
				if err != nil {
					return "", err
				}

				for _, v := range atts {
					event.AddAttachmentURL(v.url, v.mimeType)
				}
			}

			c.handleDescription(event, data, atts)
			c.handleAttendees(event, data, google)
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
		} else if !cacheRetrieved {
			cacheRetrieved = true

			startMonth, endMonth := getMonthAfterStartEndWeekDays()
			userCache, err := cachedData.getUserCache(c.userName, startMonth, endMonth)
			if err != nil {
				log.Warn().
					Err(err).
					Str("user", c.userName).
					Str("method", "getUserCache").
					Send()
				break
			}

			calData["value"] = userCache
		} else {
			break
		}
	}

	return string(cal.Serialize()), nil
}
