package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/web/templates"
)

// DashboardHandler handles the GET / dashboard route.
type DashboardHandler struct {
	c *container.Container
}

// NewDashboardHandler creates a handler using the given container.
func NewDashboardHandler(c *container.Container) *DashboardHandler {
	return &DashboardHandler{c: c}
}

// Register wires the dashboard route onto e.
func (h *DashboardHandler) Register(e *echo.Echo) {
	e.GET("/", h.index)
}

func (h *DashboardHandler) index(c echo.Context) error {
	ctx := c.Request().Context()

	if h.c.Portfolio != nil && h.c.Portfolio.HasPriceService() {
		summary, err := h.c.Portfolio.GetPortfolioSummary(ctx, true)
		if err == nil {
			groupSummary := services.ComputeGroupSummary(summary)
			return templates.DashboardPage(summary, groupSummary, nil, "").Render(ctx, c.Response().Writer)
		}
		if !errors.Is(err, services.ErrNoPriceService) {
			// Price service available but failed → show error + fallback
			groupHoldings, _ := h.c.Portfolio.GetHoldingsByGroup(ctx)
			return templates.DashboardPage(nil, nil, groupHoldings, err.Error()).Render(ctx, c.Response().Writer)
		}
	}

	// No price service or not configured → group_holdings fallback.
	if h.c.Portfolio == nil {
		return templates.DashboardPage(nil, nil, nil, "").Render(ctx, c.Response().Writer)
	}
	groupHoldings, err := h.c.Portfolio.GetHoldingsByGroup(ctx)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return templates.DashboardPage(nil, nil, groupHoldings, "").Render(ctx, c.Response().Writer)
}
