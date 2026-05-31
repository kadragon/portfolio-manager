package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/templates"
)

// GroupHandler serves the group management routes (web/routes/groups.py). The
// stock sub-routes under /groups/:id/stocks are added in Phase 2.
type GroupHandler struct {
	c *container.Container
}

// NewGroupHandler builds a GroupHandler.
func NewGroupHandler(c *container.Container) *GroupHandler {
	return &GroupHandler{c: c}
}

// Register attaches the group routes to the Echo instance.
func (h *GroupHandler) Register(e *echo.Echo) {
	e.GET("/groups", h.list)
	e.GET("/groups/:id", h.row)
	e.GET("/groups/:id/edit", h.editForm)
	e.POST("/groups", h.create)
	e.PUT("/groups/:id", h.update)
	e.DELETE("/groups/:id", h.delete)
}

func (h *GroupHandler) list(c echo.Context) error {
	groups, err := h.c.Groups.ListAll(c.Request().Context())
	if err != nil {
		return err
	}
	return render(c, templates.GroupsList(groups))
}

func (h *GroupHandler) row(c echo.Context) error {
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	g, err := h.c.Groups.GetByID(c.Request().Context(), id)
	if err != nil {
		return err
	}
	if g == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return render(c, templates.GroupRow(*g))
}

func (h *GroupHandler) editForm(c echo.Context) error {
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	g, err := h.c.Groups.GetByID(c.Request().Context(), id)
	if err != nil {
		return err
	}
	if g == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return render(c, templates.GroupForm(*g))
}

func (h *GroupHandler) create(c echo.Context) error {
	name, tp, err := groupForm(c)
	if err != nil {
		return err
	}
	g, cerr := h.c.Groups.Create(c.Request().Context(), name, tp)
	if cerr != nil {
		return cerr
	}
	return render(c, templates.GroupRow(g))
}

func (h *GroupHandler) update(c echo.Context) error {
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	name, tp, ferr := groupForm(c)
	if ferr != nil {
		return ferr
	}
	g, uerr := h.c.Groups.Update(c.Request().Context(), id, &name, &tp)
	if uerr != nil {
		return uerr
	}
	return render(c, templates.GroupRow(g))
}

func (h *GroupHandler) delete(c echo.Context) error {
	id, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	if derr := h.c.Groups.Delete(c.Request().Context(), id); derr != nil {
		return derr
	}
	return c.NoContent(http.StatusOK)
}

// groupForm parses the name/target_percentage form fields with FastAPI parity:
// name is required (422 if absent) and trimmed; target_percentage defaults to
// 0.0 when the field is absent, but a present-yet-unparseable value is 422.
func groupForm(c echo.Context) (string, float64, error) {
	if err := c.Request().ParseForm(); err != nil {
		return "", 0, echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid form")
	}
	form := c.Request().PostForm
	if !form.Has("name") {
		return "", 0, echo.NewHTTPError(http.StatusUnprocessableEntity, "name is required")
	}
	name := strings.TrimSpace(form.Get("name"))

	tp := 0.0
	if form.Has("target_percentage") {
		v, err := strconv.ParseFloat(form.Get("target_percentage"), 64)
		if err != nil {
			return "", 0, echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid target_percentage")
		}
		tp = v
	}
	return name, tp, nil
}
