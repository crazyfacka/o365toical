package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func web(url string, codeChan chan string) {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, url)
	})

	e.GET("/token", func(c echo.Context) error {
		code := c.QueryParam("code")
		codeChan <- code
		return c.NoContent(http.StatusOK)
	})

	e.Logger.Fatal(e.Start(":5000"))
}
