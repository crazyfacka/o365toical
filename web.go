package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func web(cal *Calendar) {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, cal.getURL())
	})

	e.GET("/token", func(c echo.Context) error {
		code := c.QueryParam("code")
		cal.handleToken(code)
		return c.NoContent(http.StatusOK)
	})

	e.GET("/calendar/:user", func(c echo.Context) error {
		return c.JSON(http.StatusOK, cal.getCalendar())
	})

	e.Logger.Fatal(e.Start(":5000"))
}
