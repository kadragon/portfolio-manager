package handlers_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

func setupDepositSuggestion(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewRebalanceHandler(c).Register(e)
	return e, c
}

// TestDepositSuggestionNoPriceService: POST /rebalance/deposit-suggestion with
// no price service returns the guard partial.
func TestDepositSuggestionNoPriceService(t *testing.T) {
	e, _ := setupRebalance(t)
	rec := do(e, http.MethodPost, "/rebalance/deposit-suggestion", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "가격 서비스") {
		t.Errorf("no-price-service message missing:\n%s", rec.Body.String())
	}
}

// TestDepositSuggestionInvalidInputs: bad account id or non-positive amount
// return an error message inside the partial (still 200 for HTMX swap).
func TestDepositSuggestionInvalidInputs(t *testing.T) {
	e, c := setupDepositSuggestion(t)

	rec := do(e, http.MethodPost, "/rebalance/deposit-suggestion",
		url.Values{"account_id": {"not-a-uuid"}, "amount": {"1200000"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "잘못된 계좌") {
		t.Errorf("invalid-account message missing:\n%s", rec.Body.String())
	}

	acc, err := c.Accounts.Create(context.Background(), "내 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	rec = do(e, http.MethodPost, "/rebalance/deposit-suggestion",
		url.Values{"account_id": {acc.ID.String()}, "amount": {"0"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "예치금") {
		t.Errorf("invalid-amount message missing:\n%s", rec.Body.String())
	}
}

// TestDepositSuggestionRendersResultTarget: a valid request renders the HTMX
// swap target so the dashboard form can replace it in place.
func TestDepositSuggestionRendersResultTarget(t *testing.T) {
	e, c := setupDepositSuggestion(t)

	acc, err := c.Accounts.Create(context.Background(), "내 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	rec := do(e, http.MethodPost, "/rebalance/deposit-suggestion",
		url.Values{"account_id": {acc.ID.String()}, "amount": {"1200000"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "deposit-suggestion-result") {
		t.Errorf("swap target missing:\n%s", rec.Body.String())
	}
}
