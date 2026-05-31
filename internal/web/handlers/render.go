// Package handlers contains the Echo HTTP handlers, the Go counterpart of
// web/routes/*.py. Partial vs full-page rendering is chosen by route (no
// HX-Request sniffing), matching the Python app.
package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func render(c echo.Context, comp templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(http.StatusOK)
	return comp.Render(c.Request().Context(), c.Response().Writer)
}
