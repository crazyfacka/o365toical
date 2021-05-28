package main

import (
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	cookieName = "o365toical"
)

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}

	return string(s)
}

func web() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		start := time.Now()

		cookie, err := c.Cookie(cookieName)
		if err == nil {
			if _, ok := loggedUsers[cookie.Value]; ok {
				log.Info().
					Str("src_ip", c.RealIP()).
					Str("method", c.Request().Method).
					Str("path", c.Path()).
					Int("status", http.StatusOK).
					Dur("duration", time.Since(start)).
					Msg("Session already found")

				return c.String(http.StatusOK, "https://"+c.Request().Host+"/calendar?token="+cookie.Value)
			}
		}

		cookie = new(http.Cookie)
		cookie.Name = cookieName
		cookie.Value = randomString(60)
		c.SetCookie(cookie)

		loggedUsers[cookie.Value] = newCalendarHandler()

		log.Info().
			Str("src_ip", c.RealIP()).
			Str("method", c.Request().Method).
			Str("path", c.Path()).
			Int("status", http.StatusTemporaryRedirect).
			Dur("duration", time.Since(start)).
			Msg("New session created")

		return c.Redirect(http.StatusTemporaryRedirect, loggedUsers[cookie.Value].getURL())
	})

	e.GET("/token", func(c echo.Context) error {
		start := time.Now()

		cookie, err := c.Cookie(cookieName)
		if err != nil {
			log.Error().
				Err(err).
				Str("src_ip", c.RealIP()).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Dur("duration", time.Since(start)).
				Int("status", http.StatusInternalServerError).
				Send()

			return err
		}

		code := c.QueryParam("code")

		cal := loggedUsers[cookie.Value]
		cookieToken, err := cal.handleToken(code, cookie.Value)
		if err != nil {
			log.Error().
				Err(err).
				Str("src_ip", c.RealIP()).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Dur("duration", time.Since(start)).
				Int("status", http.StatusInternalServerError).
				Send()

			return err
		}

		if cookieToken != "" && cookieToken != cookie.Value {
			cookie.Value = cookieToken
			c.SetCookie(cookie)
		}

		log.Info().
			Str("src_ip", c.RealIP()).
			Str("method", c.Request().Method).
			Str("path", c.Path()).
			Int("status", http.StatusTemporaryRedirect).
			Dur("duration", time.Since(start)).
			Msg("New token stored")

		return c.Redirect(http.StatusTemporaryRedirect, "/success")
	})

	e.GET("/success", func(c echo.Context) error {
		start := time.Now()

		cookie, err := c.Cookie(cookieName)
		if err != nil {
			log.Error().
				Err(err).
				Str("src_ip", c.RealIP()).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Dur("duration", time.Since(start)).
				Int("status", http.StatusInternalServerError).
				Send()

			return err
		}

		cachedData.storeToken(loggedUsers[cookie.Value].userName, cookie.Value)

		log.Info().
			Str("src_ip", c.RealIP()).
			Str("method", c.Request().Method).
			Str("path", c.Path()).
			Int("status", http.StatusOK).
			Dur("duration", time.Since(start)).
			Send()

		url := "https://" + c.Request().Host + "/calendar?token=" + cookie.Value

		output := `For regular devices:
` + url + `
` + url + `&full=true    # Includes tentatives and marked as 'Free' on the calendar

For Google Calendar:
` + url + `&google=true
` + url + `&google=true&full=true    # Includes tentatives and marked as 'Free' on the calendar`

		return c.String(http.StatusOK, output)
	})

	e.GET("/calendar", func(c echo.Context) error {
		start := time.Now()

		token := c.QueryParam("token")
		if len(token) == 0 {
			cookie, err := c.Cookie(cookieName)
			if err != nil {
				log.Error().
					Str("src_ip", c.RealIP()).
					Str("method", c.Request().Method).
					Str("path", c.Path()).
					Dur("duration", time.Since(start)).
					Int("status", http.StatusTemporaryRedirect).
					Msg("No token nor cookie")

				return c.Redirect(http.StatusTemporaryRedirect, "/")
			} else {
				token = cookie.Value
			}
		}

		full := false
		if c.QueryParam("full") == "true" {
			full = true
		}

		google := false
		if c.QueryParam("google") == "true" {
			google = true
		}

		cal := loggedUsers[token]
		if cal == nil {
			log.Error().
				Str("src_ip", c.RealIP()).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Dur("duration", time.Since(start)).
				Int("status", http.StatusTemporaryRedirect).
				Msg("Unknown token")

			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}

		baseHost := c.Request().Host
		if body, err := cal.getCalendar(baseHost, full, google); err == nil {
			log.Info().
				Str("src_ip", c.RealIP()).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Int("status", http.StatusOK).
				Dur("duration", time.Since(start)).
				Send()

			c.Response().Header().Set(echo.HeaderContentType, "text/calendar")
			return c.String(http.StatusOK, body)
		} else {
			log.Error().
				Err(err).
				Str("src_ip", c.RealIP()).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Dur("duration", time.Since(start)).
				Int("status", http.StatusInternalServerError).
				Send()

			return c.String(http.StatusInternalServerError, err.Error())
		}

	})

	e.GET("/attachment/:attId/:fname", func(c echo.Context) error {
		start := time.Now()

		attId, _ := url.QueryUnescape(c.Param("attId"))
		fname, _ := url.QueryUnescape(c.Param("fname"))

		baseDir := viper.GetString("attachments_dir") + "/" + attId

		log.Info().
			Str("src_ip", c.RealIP()).
			Str("method", c.Request().Method).
			Str("path", c.Path()).
			Dur("duration", time.Since(start)).
			Send()

		return c.File(baseDir + "/" + fname)
	})

	e.Logger.Fatal(e.Start(":5000"))
}
