package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func web(url string) {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, url)
	})
	e.Logger.Fatal(e.Start(":5000"))
}
