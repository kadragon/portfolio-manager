// Package handlers contains the Echo HTTP handlers, the Go counterpart of
// web/routes/*.py. Partial vs full-page rendering is chosen by route (no
// HX-Request sniffing), matching the Python app.
package handlers

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// render writes a templ component as an HTML response with the given status.
// The status is set before rendering so HTMX-driven partials can carry 422 etc.
func render(c echo.Context, status int, comp templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(status)
	return comp.Render(c.Request().Context(), c.Response().Writer)
}
