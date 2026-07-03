package handlers

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
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
	e.POST("/rebalance/deposit-suggestion", h.depositSuggestion)
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

	summary, plan, err := h.buildPlan(c, restrictOverseas)
	if err != nil {
		return templates.RebalancePage(nil, nil, nil, restrictOverseas, err.Error(), false).Render(ctx, c.Response().Writer)
	}

	groupSummary := services.ComputeGroupSummary(summary)
	return templates.RebalancePage(summary, groupSummary, plan, restrictOverseas, "", h.c.OrderClient != nil).Render(ctx, c.Response().Writer)
}

func (h *RebalanceHandler) execute(c echo.Context) error {
	ctx := c.Request().Context()
	restrictOverseas := c.FormValue("restrict_overseas") != ""

	if !h.c.Portfolio.HasPriceService() {
		return templates.RebalanceResultPartial(nil, "가격 서비스 없음", false).Render(ctx, c.Response().Writer)
	}

	_, plan, err := h.buildPlan(c, restrictOverseas)
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

	accountIDParam := c.FormValue("account_id")
	if accountIDParam != "" {
		accountID, err := uuidx.Parse(accountIDParam)
		if err != nil {
			return templates.RebalanceResultPartial(nil, "잘못된 계좌 ID입니다.", false).Render(ctx, c.Response().Writer)
		}
		allRecs = filterRecsByAccount(allRecs, accountID)
		if len(allRecs) == 0 {
			return templates.RebalanceResultPartial(nil, "해당 계좌에 실행할 주문이 없습니다.", false).Render(ctx, c.Response().Writer)
		}
	}

	result := h.c.RebalanceExecution.ExecuteRebalanceOrders(allRecs, false, exchangeMap)
	if err := templates.RebalanceResultPartial(&result, "주문 실행 완료", true).Render(ctx, c.Response().Writer); err != nil {
		return err
	}
	if accountIDParam != "" {
		return templates.RebalanceAccountExecuteDoneOOB(accountIDParam).Render(ctx, c.Response().Writer)
	}
	return nil
}

// filterRecsByAccount narrows recs to a single account for per-account execution.
func filterRecsByAccount(recs []models.RebalanceRecommendation, accountID uuidx.UUID) []models.RebalanceRecommendation {
	out := make([]models.RebalanceRecommendation, 0, len(recs))
	for _, r := range recs {
		if r.AccountID == accountID {
			out = append(out, r)
		}
	}
	return out
}

// depositSuggestion renders the deposit allocation suggestion partial: it
// simulates depositing the given cash amount into the given account and shows
// the buy recommendations that deposit funds (cash-flow rebalancing).
func (h *RebalanceHandler) depositSuggestion(c echo.Context) error {
	ctx := c.Request().Context()
	render := func(recs []models.RebalanceRecommendation, hasSells bool, errMsg string) error {
		return templates.DepositSuggestionPartial(recs, hasSells, errMsg).Render(ctx, c.Response().Writer)
	}

	if !h.c.Portfolio.HasPriceService() {
		return render(nil, false, "가격 서비스가 설정되지 않았습니다. KIS API 키를 확인하세요.")
	}
	accountID, err := uuidx.Parse(c.FormValue("account_id"))
	if err != nil {
		return render(nil, false, "잘못된 계좌 ID입니다.")
	}
	amount, err := numeric.FromString(strings.TrimSpace(c.FormValue("amount")))
	if err != nil || !amount.IsPositive() {
		return render(nil, false, "예치금은 0보다 큰 숫자여야 합니다.")
	}

	_, params, err := h.assemblePlanParams(c, false)
	if err != nil {
		return render(nil, false, err.Error())
	}
	plan, err := h.c.Rebalance.BuildDepositPlan(params, accountID, amount.Decimal)
	if err != nil {
		return render(nil, false, err.Error())
	}
	// Only the deposited account's buys are actionable from this deposit;
	// plan.SellRecs non-empty means a band breach needs the full rebalance view.
	return render(filterRecsByAccount(plan.BuyRecs, accountID), len(plan.SellRecs) > 0, "")
}

// assemblePlanParams gathers BuildPlanParams (summary, accounts, holdings,
// groups, stocks) shared by plan-building endpoints.
func (h *RebalanceHandler) assemblePlanParams(c echo.Context, restrictOverseas bool) (*models.PortfolioSummary, services.BuildPlanParams, error) {
	ctx := c.Request().Context()

	summary, err := h.c.Portfolio.GetPortfolioSummary(ctx, false)
	if err != nil {
		return nil, services.BuildPlanParams{}, err
	}

	allAccounts, err := h.c.Accounts.ListAll(ctx)
	if err != nil {
		return nil, services.BuildPlanParams{}, err
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

	return summary, services.BuildPlanParams{
		Summary:           *summary,
		Accounts:          allAccounts,
		HoldingsByAccount: holdingsByAccount,
		Groups:            allGroups,
		Stocks:            allStocks,
		RestrictOverseas:  restrictOverseas,
	}, nil
}

// buildPlan computes the rebalance plan and returns the portfolio summary it was
// built from, so callers can reuse the summary without re-querying.
func (h *RebalanceHandler) buildPlan(c echo.Context, restrictOverseas bool) (*models.PortfolioSummary, *models.RebalancePlan, error) {
	summary, params, err := h.assemblePlanParams(c, restrictOverseas)
	if err != nil {
		return nil, nil, err
	}
	plan, err := h.c.Rebalance.BuildPlan(params)
	if err != nil {
		return nil, nil, err
	}
	return summary, &plan, nil
}
