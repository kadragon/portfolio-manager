package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/templates"
)

// HoldingHandler handles /accounts/:id/holdings routes.
type HoldingHandler struct {
	c *container.Container
}

// NewHoldingHandler creates a handler using the given container.
func NewHoldingHandler(c *container.Container) *HoldingHandler {
	return &HoldingHandler{c: c}
}

// Register wires all holding routes onto e.
func (h *HoldingHandler) Register(e *echo.Echo) {
	e.GET("/accounts/:id/holdings", h.list)
	e.PUT("/accounts/:id/holdings/bulk", h.bulkUpdate) // static before :hid param
	e.POST("/accounts/:id/holdings/by-ticker", h.createByTicker)
	e.POST("/accounts/:id/holdings", h.create)
	e.GET("/accounts/:id/holdings/:hid", h.row)
	e.GET("/accounts/:id/holdings/:hid/edit", h.editForm)
	e.PUT("/accounts/:id/holdings/:hid", h.update)
	e.DELETE("/accounts/:id/holdings/:hid", h.deleteHolding)
}

func (h *HoldingHandler) list(c echo.Context) error {
	ctx := c.Request().Context()
	aid, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	account, err := h.c.Accounts.GetByID(ctx, aid)
	if err != nil {
		return err
	}
	if account == nil {
		return c.NoContent(http.StatusNotFound)
	}
	holdings, err := h.c.Holdings.ListByAccount(ctx, aid)
	if err != nil {
		return err
	}
	allStocks, err := h.c.Stocks.ListAll(ctx)
	if err != nil {
		return err
	}
	stockMap := makeStockMap(allStocks)
	stockNameMap := templates.BuildStockNameMap(allStocks)
	allGroups, err := h.c.Groups.ListAll(ctx)
	if err != nil {
		return err
	}
	return templates.HoldingsPage(*account, holdings, stockMap, stockNameMap, allStocks, allGroups).
		Render(ctx, c.Response().Writer)
}

func (h *HoldingHandler) create(c echo.Context) error {
	ctx := c.Request().Context()
	aid, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	sid, err := uuidx.Parse(c.FormValue("stock_id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	qty := parseQuantity(c.FormValue("quantity"))
	if _, err := h.c.Holdings.Create(ctx, aid, sid, qty); err != nil {
		return err
	}
	return h.renderHoldingsRows(c, aid)
}

func (h *HoldingHandler) bulkUpdate(c echo.Context) error {
	ctx := c.Request().Context()
	aid, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	if err := c.Request().ParseForm(); err != nil {
		return err
	}
	holdingIDStrs := c.Request().PostForm["holding_id"]
	quantityStrs := c.Request().PostForm["quantity"]

	errorResult := func(msg string) error {
		c.Response().WriteHeader(http.StatusBadRequest)
		return templates.HoldingsBulkResult(false, msg, nil, nil, nil, aid).
			Render(ctx, c.Response().Writer)
	}

	if len(holdingIDStrs) == 0 {
		return errorResult("요청 payload에 holding_id가 없습니다.")
	}
	if len(holdingIDStrs) != len(quantityStrs) {
		return errorResult("holding_id와 quantity 개수가 일치하지 않습니다.")
	}

	// Check duplicates
	seen := make(map[string]struct{}, len(holdingIDStrs))
	for _, s := range holdingIDStrs {
		if _, dup := seen[s]; dup {
			return errorResult("요청에 중복된 holding_id가 포함되어 있습니다.")
		}
		seen[s] = struct{}{}
	}

	// Parse and validate
	threshold, _ := decimal.NewFromString("0.000001")
	updates := make([]repositories.HoldingUpdate, 0, len(holdingIDStrs))
	for i, idStr := range holdingIDStrs {
		hid, err := uuidx.Parse(idStr)
		if err != nil {
			return errorResult("holding_id 형식이 올바르지 않습니다.")
		}
		qty := parseQuantity(quantityStrs[i])
		if qty.Decimal.LessThan(threshold) {
			return errorResult("모든 수량은 0보다 커야 합니다.")
		}
		updates = append(updates, repositories.HoldingUpdate{ID: hid, Quantity: qty})
	}

	if err := h.c.Holdings.BulkUpdateByAccount(ctx, aid, updates); err != nil {
		msg := normalizeBulkError(err.Error())
		return errorResult(msg)
	}

	holdings, err := h.c.Holdings.ListByAccount(ctx, aid)
	if err != nil {
		return err
	}
	allStocks, err := h.c.Stocks.ListAll(ctx)
	if err != nil {
		return err
	}
	stockMap := makeStockMap(allStocks)
	stockNameMap := templates.BuildStockNameMap(allStocks)
	return templates.HoldingsBulkResult(true, "보유 수량을 일괄 저장했습니다.", holdings, stockMap, stockNameMap, aid).
		Render(ctx, c.Response().Writer)
}

func (h *HoldingHandler) createByTicker(c echo.Context) error {
	ctx := c.Request().Context()
	aid, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	errHeaders := map[string]string{
		"HX-Retarget": "#by-ticker-error",
		"HX-Reswap":   "innerHTML",
	}
	tickerError := func(msg string) error {
		for k, v := range errHeaders {
			c.Response().Header().Set(k, v)
		}
		c.Response().WriteHeader(http.StatusUnprocessableEntity)
		_, _ = c.Response().Write([]byte(msg))
		return nil
	}

	normalizedTicker := strings.TrimSpace(strings.ToUpper(c.FormValue("ticker")))
	if normalizedTicker == "" {
		return tickerError("티커를 입력하세요.")
	}

	stock, err := h.c.Stocks.GetByTicker(ctx, normalizedTicker)
	if err != nil {
		return err
	}

	if stock == nil {
		groups, err := h.c.Groups.ListAll(ctx)
		if err != nil {
			return err
		}

		rawGroupID := strings.TrimSpace(c.FormValue("group_id"))
		var targetGroupID *uuidx.UUID

		if rawGroupID != "" {
			gid, err := uuidx.Parse(rawGroupID)
			if err != nil {
				return tickerError("선택한 그룹이 올바르지 않습니다.")
			}
			found := false
			for _, g := range groups {
				if g.ID == gid {
					found = true
					break
				}
			}
			if !found {
				return tickerError("선택한 그룹을 찾을 수 없습니다.")
			}
			targetGroupID = &gid
		} else if len(groups) == 0 {
			groupName := strings.TrimSpace(c.FormValue("new_group_name"))
			if groupName == "" {
				return tickerError("그룹이 없어 새 그룹 이름이 필요합니다.")
			}
			created, err := h.c.Groups.Create(ctx, groupName, 0.0)
			if err != nil {
				return err
			}
			targetGroupID = &created.ID
		} else {
			return tickerError("새 티커는 그룹을 선택해야 합니다.")
		}

		newStock, err := h.c.Stocks.Create(ctx, normalizedTicker, *targetGroupID)
		if err != nil {
			return err
		}
		stock = &newStock
	}

	qty := parseQuantity(c.FormValue("quantity"))
	if _, err := h.c.Holdings.Create(ctx, aid, stock.ID, qty); err != nil {
		return err
	}
	return h.renderHoldingsRows(c, aid)
}

func (h *HoldingHandler) row(c echo.Context) error {
	ctx := c.Request().Context()
	aid, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	hid, err := uuidx.Parse(c.Param("hid"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	holding, err := h.c.Holdings.GetByID(ctx, hid)
	if err != nil {
		return err
	}
	if holding == nil || holding.AccountID != aid {
		return c.NoContent(http.StatusNotFound)
	}
	allStocks, err := h.c.Stocks.ListAll(ctx)
	if err != nil {
		return err
	}
	stockMap := makeStockMap(allStocks)
	stockNameMap := templates.BuildStockNameMap(allStocks)
	return templates.HoldingRow(*holding, stockMap, stockNameMap, aid).
		Render(ctx, c.Response().Writer)
}

func (h *HoldingHandler) editForm(c echo.Context) error {
	ctx := c.Request().Context()
	aid, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	hid, err := uuidx.Parse(c.Param("hid"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	holding, err := h.c.Holdings.GetByID(ctx, hid)
	if err != nil {
		return err
	}
	if holding == nil || holding.AccountID != aid {
		return c.NoContent(http.StatusNotFound)
	}
	allStocks, err := h.c.Stocks.ListAll(ctx)
	if err != nil {
		return err
	}
	stockMap := makeStockMap(allStocks)
	stockNameMap := templates.BuildStockNameMap(allStocks)
	return templates.HoldingForm(*holding, stockMap, stockNameMap, aid).
		Render(ctx, c.Response().Writer)
}

func (h *HoldingHandler) update(c echo.Context) error {
	ctx := c.Request().Context()
	aid, err := uuidx.Parse(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	hid, err := uuidx.Parse(c.Param("hid"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	qty := parseQuantity(c.FormValue("quantity"))
	holding, err := h.c.Holdings.Update(ctx, hid, qty)
	if err != nil {
		return err
	}
	allStocks, err := h.c.Stocks.ListAll(ctx)
	if err != nil {
		return err
	}
	stockMap := makeStockMap(allStocks)
	stockNameMap := templates.BuildStockNameMap(allStocks)
	return templates.HoldingRow(holding, stockMap, stockNameMap, aid).
		Render(ctx, c.Response().Writer)
}

func (h *HoldingHandler) deleteHolding(c echo.Context) error {
	ctx := c.Request().Context()
	hid, err := uuidx.Parse(c.Param("hid"))
	if err != nil {
		return c.NoContent(http.StatusUnprocessableEntity)
	}
	if err := h.c.Holdings.Delete(ctx, hid); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (h *HoldingHandler) renderHoldingsRows(c echo.Context, accountID uuidx.UUID) error {
	ctx := c.Request().Context()
	holdings, err := h.c.Holdings.ListByAccount(ctx, accountID)
	if err != nil {
		return err
	}
	allStocks, err := h.c.Stocks.ListAll(ctx)
	if err != nil {
		return err
	}
	stockMap := makeStockMap(allStocks)
	stockNameMap := templates.BuildStockNameMap(allStocks)
	return templates.HoldingsRows(holdings, stockMap, stockNameMap, accountID).
		Render(ctx, c.Response().Writer)
}

func makeStockMap(stocks []models.Stock) map[uuidx.UUID]models.Stock {
	m := make(map[uuidx.UUID]models.Stock, len(stocks))
	for _, s := range stocks {
		m[s.ID] = s
	}
	return m
}

func parseQuantity(s string) numeric.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return numeric.Zero
	}
	return numeric.Wrap(d)
}

func normalizeBulkError(msg string) string {
	known := map[string]string{
		"all holdings must belong to account":        "요청한 holding_id가 현재 계좌에 속하지 않습니다.",
		"duplicate holding_ids are not allowed":      "요청에 중복된 holding_id가 포함되어 있습니다.",
		"quantity must be greater than zero":         "모든 수량은 0보다 커야 합니다.",
		"holding_ids and quantities length mismatch": "holding_id와 quantity 개수가 일치하지 않습니다.",
		"holding_ids and quantities are required":    "holding_id와 quantity가 모두 필요합니다.",
		"선택한 보유 내역이 해당 계좌에 속하지 않습니다.":                "요청한 holding_id가 현재 계좌에 속하지 않습니다.",
	}
	if m, ok := known[msg]; ok {
		return m
	}
	if msg == "" {
		return "보유 수량 일괄 저장 중 오류가 발생했습니다."
	}
	return msg
}
