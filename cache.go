package main

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
)

func getCalendarMonthForUser(c *Calendar, start time.Time, end time.Time) ([]interface{}, error) {
	var calData map[string]interface{}
	var cachedValues []interface{}

	url := "https://graph.microsoft.com/v1.0/me/calendarview?startdatetime=" + start.Format(RFC3339Short) + "&enddatetime=" + end.Format(RFC3339Short) + "&top=10&skip=0"
	body, err := c.getRemoteData(url)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &calData); err != nil {
		return nil, err
	}

	for {
		values := calData["value"].([]interface{})
		cachedValues = append(cachedValues, values...)

		if nextPage, ok := calData["@odata.nextLink"].(string); ok {
			calData = make(map[string]interface{})
			body, err := c.getRemoteData(nextPage)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(body, &calData); err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	return cachedValues, nil
}

func refreshCache() {
	for {
		time.Sleep(1 * time.Second)
		for _, v := range loggedUsers {
			start, end := getMonthAfterStartEndWeekDays()

			lastUpdated, err := cachedData.getCacheForUserLastUpdate(v.userName, start, end)
			if err != nil || time.Since(lastUpdated).Hours() >= 24 {
				cachedValues, err := getCalendarMonthForUser(v, start, end)
				if err != nil {
					log.Error().
						Err(err).
						Str("user", v.userName).
						Str("method", "getCalendarMonthForUser").
						Send()

					continue
				}

				err = cachedData.saveCacheForUser(v.userName, start, end, cachedValues)
				if err != nil {
					log.Error().
						Err(err).
						Str("user", v.userName).
						Str("method", "saveCacheForUser").
						Send()
				}
			}
		}
	}
}
