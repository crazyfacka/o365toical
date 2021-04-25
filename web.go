package main

import (
	"math/rand"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	cookieName = "o365toical"
)

var loggedUsers map[string]*Calendar

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

	loggedUsers = make(map[string]*Calendar)

	e.GET("/", func(c echo.Context) error {
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			cookie = new(http.Cookie)
			cookie.Name = cookieName
			cookie.Value = randomString(60)
			c.SetCookie(cookie)

			loggedUsers[cookie.Value] = newCalendarHandler()
		}

		cal := loggedUsers[cookie.Value]
		return c.Redirect(http.StatusTemporaryRedirect, cal.getURL())
	})

	e.GET("/token", func(c echo.Context) error {
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			return err
		}

		code := c.QueryParam("code")

		cal := loggedUsers[cookie.Value]
		if err = cal.handleToken(code); err != nil {
			return err
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/success")
	})

	e.GET("/success", func(c echo.Context) error {
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			return err
		}

		return c.String(http.StatusOK, "https://"+c.Request().Host+"/calendar?token="+cookie.Value)
	})

	e.GET("/calendar", func(c echo.Context) error {
		var token string

		cookie, err := c.Cookie(cookieName)
		if err != nil {
			token = c.QueryParam("token")
		} else {
			token = cookie.Value
		}

		cal := loggedUsers[token]

		return c.JSON(http.StatusOK, cal.getCalendar())
	})

	e.Logger.Fatal(e.Start(":5000"))
}
