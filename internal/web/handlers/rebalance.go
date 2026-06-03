package handlers

import (
	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/templates"
)

// RebalanceHandler handles GET /rebalance and POST /rebalance/execute.
type RebalanceHandler struct {
	c *container.Container
}

// NewRebalanceHandler creates the handler.
func NewRebalanceHandler(c *container.Container) *RebalanceHandler {
	return &RebalanceHandler{c: c}
}

// Register wires the rebalance routes onto e.
func (h *RebalanceHandler) Register(e *echo.Echo) {
	e.GET("/rebalance", h.view)
	e.POST("/rebalance/execute", h.execute)
}

func (h *RebalanceHandler) view(c echo.Context) error {
	ctx := c.Request().Context()
	restrictOverseas := c.QueryParam("restrict_overseas") != ""

	if !h.c.Portfolio.HasPriceService() {
		return templates.RebalancePage(
			nil, nil, nil, restrictOverseas,
			"가격 서비스가 설정되지 않았습니다. KIS API 키를 확인하세요.",
			h.c.OrderClient != nil,
		).Render(ctx, c.Response().Writer)
	}

	plan, err := h.buildPlan(c, restrictOverseas)
	if err != nil {
		return templates.RebalancePage(nil, nil, nil, restrictOverseas, err.Error(), false).Render(ctx, c.Response().Writer)
	}

	summary, _ := h.c.Portfolio.GetPortfolioSummary(ctx, false)
	var groupSummary []models.GroupSummaryRow
	if summary != nil {
		groupSummary = services.ComputeGroupSummary(summary)
	}
	return templates.RebalancePage(summary, groupSummary, plan, restrictOverseas, "", h.c.OrderClient != nil).Render(ctx, c.Response().Writer)
}

func (h *RebalanceHandler) execute(c echo.Context) error {
	ctx := c.Request().Context()
	restrictOverseas := c.FormValue("restrict_overseas") != ""

	if !h.c.Portfolio.HasPriceService() {
		return templates.RebalanceResultPartial(nil, "가격 서비스 없음", false).Render(ctx, c.Response().Writer)
	}

	plan, err := h.buildPlan(c, restrictOverseas)
	if err != nil {
		return templates.RebalanceResultPartial(nil, err.Error(), false).Render(ctx, c.Response().Writer)
	}

	allStocks, _ := h.c.Stocks.ListAll(ctx)
	exchangeMap := map[string]string{}
	for _, s := range allStocks {
		if s.Exchange != nil && *s.Exchange != "" {
			exchangeMap[s.Ticker] = *s.Exchange
		}
	}

	allRecs := append(plan.SellRecs, plan.BuyRecs...)
	result := h.c.RebalanceExecution.ExecuteRebalanceOrders(allRecs, false, exchangeMap)
	return templates.RebalanceResultPartial(&result, "주문 실행 완료", true).Render(ctx, c.Response().Writer)
}

func (h *RebalanceHandler) buildPlan(c echo.Context, restrictOverseas bool) (*models.RebalancePlan, error) {
	ctx := c.Request().Context()

	summary, err := h.c.Portfolio.GetPortfolioSummary(ctx, false)
	if err != nil {
		return nil, err
	}

	allAccounts, err := h.c.Accounts.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	holdingsByAccount := make(map[uuidx.UUID][]models.Holding, len(allAccounts))
	for _, acc := range allAccounts {
		hs, err := h.c.Holdings.ListByAccount(ctx, acc.ID)
		if err != nil {
			continue
		}
		holdingsByAccount[acc.ID] = hs
	}

	allGroups, _ := h.c.Groups.ListAll(ctx)
	allStocks, _ := h.c.Stocks.ListAll(ctx)

	plan, err := h.c.Rebalance.BuildPlan(services.BuildPlanParams{
		Summary:           *summary,
		Accounts:          allAccounts,
		HoldingsByAccount: holdingsByAccount,
		Groups:            allGroups,
		Stocks:            allStocks,
		RestrictOverseas:  restrictOverseas,
	})
	if err != nil {
		return nil, err
	}
	return &plan, nil
}
