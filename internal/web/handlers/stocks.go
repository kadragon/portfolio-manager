package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/templates"
)

// StockHandler serves the stock sub-routes under /groups/:group_id/stocks.
type StockHandler struct {
	c *container.Container
}

// NewStockHandler builds a StockHandler.
func NewStockHandler(c *container.Container) *StockHandler {
	return &StockHandler{c: c}
}

// assetClassEquals reports whether the stored (nullable) asset class equals
// want, treating a nil class as equal to the empty (unclassified) value.
func assetClassEquals(current *string, want string) bool {
	if current == nil {
		return want == ""
	}
	return *current == want
}

// Register attaches the stock routes to the Echo instance.
func (h *StockHandler) Register(e *echo.Echo) {
	e.GET("/groups/:group_id/stocks", h.list)
	e.POST("/groups/:group_id/stocks", h.create)
	e.GET("/groups/:group_id/stocks/:stock_id", h.row)
	e.GET("/groups/:group_id/stocks/:stock_id/edit", h.editForm)
	e.PUT("/groups/:group_id/stocks/:stock_id", h.update)
	e.DELETE("/groups/:group_id/stocks/:stock_id", h.delete)
}

func (h *StockHandler) list(c echo.Context) error {
	groupID, err := uuidx.Parse(c.Param("group_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	ctx := c.Request().Context()
	g, err := h.c.Groups.GetByID(ctx, groupID)
	if err != nil {
		return err
	}
	if g == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	stocks, err := h.c.Stocks.ListByGroup(ctx, groupID)
	if err != nil {
		return err
	}
	return render(c, templates.StocksPage(*g, stocks))
}

func (h *StockHandler) create(c echo.Context) error {
	groupID, err := uuidx.Parse(c.Param("group_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	if err := c.Request().ParseForm(); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid form")
	}
	form := c.Request().PostForm
	if !form.Has("ticker") {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "ticker is required")
	}
	ticker := strings.ToUpper(strings.TrimSpace(form.Get("ticker")))
	s, err := h.c.Stocks.Create(c.Request().Context(), ticker, groupID)
	if err != nil {
		return err
	}
	return render(c, templates.StockRow(s, groupID))
}

func (h *StockHandler) row(c echo.Context) error {
	groupID, err := uuidx.Parse(c.Param("group_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	stockID, err := uuidx.Parse(c.Param("stock_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid stock id")
	}
	s, err := h.c.Stocks.GetByID(c.Request().Context(), stockID)
	if err != nil {
		return err
	}
	if s == nil || s.GroupID != groupID {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return render(c, templates.StockRow(*s, groupID))
}

func (h *StockHandler) editForm(c echo.Context) error {
	groupID, err := uuidx.Parse(c.Param("group_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	stockID, err := uuidx.Parse(c.Param("stock_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid stock id")
	}
	ctx := c.Request().Context()
	s, err := h.c.Stocks.GetByID(ctx, stockID)
	if err != nil {
		return err
	}
	if s == nil || s.GroupID != groupID {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	groups, err := h.c.Groups.ListAll(ctx)
	if err != nil {
		return err
	}
	return render(c, templates.StockForm(*s, groupID, groups))
}

func (h *StockHandler) update(c echo.Context) error {
	groupID, err := uuidx.Parse(c.Param("group_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid group id")
	}
	stockID, err := uuidx.Parse(c.Param("stock_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid stock id")
	}
	ctx := c.Request().Context()
	s, err := h.c.Stocks.GetByID(ctx, stockID)
	if err != nil {
		return err
	}
	if s == nil || s.GroupID != groupID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := c.Request().ParseForm(); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid form")
	}
	form := c.Request().PostForm
	if !form.Has("ticker") {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "ticker is required")
	}
	ticker := strings.ToUpper(strings.TrimSpace(form.Get("ticker")))
	if ticker == "" {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "ticker cannot be empty")
	}

	destGroupID := s.GroupID
	rawTarget := strings.TrimSpace(form.Get("target_group_id"))
	if rawTarget != "" {
		parsed, perr := uuidx.Parse(rawTarget)
		if perr != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid target_group_id")
		}
		groups, gerr := h.c.Groups.ListAll(ctx)
		if gerr != nil {
			return gerr
		}
		found := false
		for _, g := range groups {
			if g.ID == parsed {
				found = true
				break
			}
		}
		if !found {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		destGroupID = parsed
	}

	updated := *s
	if destGroupID != s.GroupID {
		upd, uerr := h.c.Stocks.UpdateGroup(ctx, s.ID, destGroupID)
		if uerr != nil {
			return uerr
		}
		updated = upd
	}
	if ticker != updated.Ticker {
		upd, uerr := h.c.Stocks.UpdateTicker(ctx, s.ID, ticker)
		if uerr != nil {
			return uerr
		}
		updated = upd
	}
	// asset_class: "etf" / "stock", or empty to clear ("미분류", which re-enables
	// classification on the next sync). The "unknown" sentinel is set only by the
	// classifier — not accepted here, so a client POST cannot force it; other
	// values leave it unchanged.
	if form.Has("asset_class") {
		assetClass := strings.TrimSpace(form.Get("asset_class"))
		if (assetClass == "" || models.ValidAssetClass(assetClass)) && !assetClassEquals(updated.AssetClass, assetClass) {
			upd, uerr := h.c.Stocks.UpdateAssetClass(ctx, s.ID, assetClass)
			if uerr != nil {
				return uerr
			}
			updated = upd
		}
	}
	// security_group: KIS scty_grp_id_cd (e.g. "EF"/"ST"), normalized uppercase;
	// empty clears it back to "미지정". Only recognized codes are accepted;
	// KIS sync bypasses this handler and writes codes directly.
	if form.Has("security_group") {
		securityGroup := strings.ToUpper(strings.TrimSpace(form.Get("security_group")))
		if (securityGroup == "" || models.ValidSecurityGroup(securityGroup)) &&
			!assetClassEquals(updated.SecurityGroup, securityGroup) {
			upd, uerr := h.c.Stocks.UpdateSecurityGroup(ctx, s.ID, securityGroup)
			if uerr != nil {
				return uerr
			}
			updated = upd
		}
	}
	return render(c, templates.StockRow(updated, updated.GroupID))
}

func (h *StockHandler) delete(c echo.Context) error {
	stockID, err := uuidx.Parse(c.Param("stock_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid stock id")
	}
	if err := h.c.Stocks.Delete(c.Request().Context(), stockID); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}
