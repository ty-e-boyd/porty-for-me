package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ty-e-boyd/porty-for-me/template"
)

func main() {
	e := echo.New()

	// Little bit of middlewares for housekeeping
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())

	// This will initiate our template renderer
	template.NewTemplateRenderer(e, "public/*.html")
	e.GET("/", func(e echo.Context) error {
		return e.Render(http.StatusOK, "index", nil)
	})

	e.Logger.Fatal(e.Start(":4040"))
}
