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
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

func setupHoldings(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewAccountHandler(c).Register(e)
	handlers.NewHoldingHandler(c).Register(e)
	return e, c
}

func seedHoldingH(t *testing.T, c *container.Container, accountID, stockID uuidx.UUID, qty string) models.Holding {
	t.Helper()
	q, _ := numeric.FromString(qty)
	h, err := c.Holdings.Create(context.Background(), accountID, stockID, q)
	if err != nil {
		t.Fatalf("seed holding: %v", err)
	}
	return h
}

// --- list ---

func TestHoldingsListOK(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "내 계좌")
	g := seedGroup(t, c, "그룹A")
	s := seedStock(t, c, "005930", g)
	seedHoldingH(t, c, a.ID, s.ID, "10")

	rec := do(e, http.MethodGet, "/accounts/"+a.ID.String()+"/holdings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "005930") {
		t.Error("ticker missing from holdings page")
	}
	if !strings.Contains(body, "내 계좌 보유 내역") {
		t.Error("account name missing from page title")
	}
}

func TestHoldingsListNotFound(t *testing.T) {
	e, _ := setupHoldings(t)
	rec := do(e, http.MethodGet, "/accounts/00000000000000000000000000000000/holdings", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

// --- create ---

func TestHoldingCreateRendersRows(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "AAPL", g)

	rec := do(e, http.MethodPost, "/accounts/"+a.ID.String()+"/holdings", url.Values{
		"stock_id": {s.ID.String()},
		"quantity": {"5.5"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "AAPL") {
		t.Error("ticker missing from created row")
	}
}

// --- row ---

func TestHoldingRowOK(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "TSLA", g)
	h := seedHoldingH(t, c, a.ID, s.ID, "3")

	rec := do(e, http.MethodGet, "/accounts/"+a.ID.String()+"/holdings/"+h.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "TSLA") {
		t.Error("ticker missing from row")
	}
}

func TestHoldingRowWrongAccount(t *testing.T) {
	e, c := setupHoldings(t)
	a1 := seedAccount(t, c, "계좌1")
	a2 := seedAccount(t, c, "계좌2")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "MSFT", g)
	h := seedHoldingH(t, c, a1.ID, s.ID, "1")

	rec := do(e, http.MethodGet, "/accounts/"+a2.ID.String()+"/holdings/"+h.ID.String(), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- edit form ---

func TestHoldingEditFormOK(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "NVDA", g)
	h := seedHoldingH(t, c, a.ID, s.ID, "2")

	rec := do(e, http.MethodGet, "/accounts/"+a.ID.String()+"/holdings/"+h.ID.String()+"/edit", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "NVDA") {
		t.Error("ticker missing from edit form")
	}
	if !strings.Contains(body, "저장") {
		t.Error("submit button missing")
	}
}

// --- update ---

func TestHoldingUpdateOK(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "AMZN", g)
	h := seedHoldingH(t, c, a.ID, s.ID, "1")

	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String()+"/holdings/"+h.ID.String(), url.Values{
		"quantity": {"7.25"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "AMZN") {
		t.Error("ticker missing from updated row")
	}
}

// --- delete ---

func TestHoldingDeleteOK(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "META", g)
	h := seedHoldingH(t, c, a.ID, s.ID, "4")

	rec := do(e, http.MethodDelete, "/accounts/"+a.ID.String()+"/holdings/"+h.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	all, _ := c.Holdings.ListByAccount(context.Background(), a.ID)
	if len(all) != 0 {
		t.Fatalf("holding not deleted, count=%d", len(all))
	}
}

// --- bulk update ---

func TestHoldingBulkUpdateOK(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "GOOG", g)
	h := seedHoldingH(t, c, a.ID, s.ID, "1")

	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String()+"/holdings/bulk", url.Values{
		"holding_id": {h.ID.String()},
		"quantity":   {"99"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "일괄 저장했습니다") {
		t.Error("success message missing")
	}
}

func TestHoldingBulkUpdateEmptyIDs400(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")

	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String()+"/holdings/bulk", url.Values{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "holding_id가 없습니다") {
		t.Error("error message missing")
	}
}

func TestHoldingBulkUpdateLengthMismatch400(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "IBM", g)
	h := seedHoldingH(t, c, a.ID, s.ID, "1")

	// holding_id without corresponding quantity
	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String()+"/holdings/bulk", url.Values{
		"holding_id": {h.ID.String()},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHoldingBulkUpdateInvalidQty400(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "NFLX", g)
	h := seedHoldingH(t, c, a.ID, s.ID, "1")

	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String()+"/holdings/bulk", url.Values{
		"holding_id": {h.ID.String()},
		"quantity":   {"0"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "0보다 커야") {
		t.Errorf("qty error message missing, body=%s", rec.Body.String())
	}
}

// --- by-ticker ---

func TestHoldingCreateByTickerExistingStock(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "그룹")
	seedStock(t, c, "AAPL", g)

	rec := do(e, http.MethodPost, "/accounts/"+a.ID.String()+"/holdings/by-ticker", url.Values{
		"ticker":   {"AAPL"},
		"quantity": {"3"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "AAPL") {
		t.Error("ticker missing")
	}
}

func TestHoldingCreateByTickerNewStockWithGroup(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	g := seedGroup(t, c, "신규그룹")

	rec := do(e, http.MethodPost, "/accounts/"+a.ID.String()+"/holdings/by-ticker", url.Values{
		"ticker":   {"NEW1"},
		"quantity": {"1"},
		"group_id": {g.ID.String()},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "NEW1") {
		t.Error("new ticker missing")
	}
}

func TestHoldingCreateByTickerNewGroupAutoCreate(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	// No groups at all — auto-create group with new_group_name

	rec := do(e, http.MethodPost, "/accounts/"+a.ID.String()+"/holdings/by-ticker", url.Values{
		"ticker":         {"AUTO1"},
		"quantity":       {"1"},
		"new_group_name": {"자동생성그룹"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "AUTO1") {
		t.Error("ticker missing after auto group create")
	}
}

func TestHoldingCreateByTickerEmptyTicker422(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")

	rec := do(e, http.MethodPost, "/accounts/"+a.ID.String()+"/holdings/by-ticker", url.Values{
		"ticker":   {""},
		"quantity": {"1"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if rec.Header().Get("HX-Retarget") != "#by-ticker-error" {
		t.Error("HX-Retarget header missing")
	}
}

// TestHoldingBulkUpdateWrongAccountError exercises normalizeBulkError by submitting
// a holding_id that belongs to a different account.
func TestHoldingBulkUpdateWrongAccountError(t *testing.T) {
	e, c := setupHoldings(t)
	a1 := seedAccount(t, c, "계좌1")
	a2 := seedAccount(t, c, "계좌2")
	g := seedGroup(t, c, "그룹")
	s := seedStock(t, c, "TST1", g)
	h2 := seedHoldingH(t, c, a2.ID, s.ID, "5") // belongs to a2

	// PUT /accounts/a1/holdings/bulk with a holding from a2 → normalizeBulkError called
	rec := do(e, http.MethodPut, "/accounts/"+a1.ID.String()+"/holdings/bulk", url.Values{
		"holding_id": {h2.ID.String()},
		"quantity":   {"10"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "holding_id") && !strings.Contains(rec.Body.String(), "계좌") {
		t.Errorf("error message unexpected:\n%s", rec.Body.String())
	}
}

func TestHoldingCreateByTickerNoGroupError422(t *testing.T) {
	e, c := setupHoldings(t)
	a := seedAccount(t, c, "계좌")
	seedGroup(t, c, "기존그룹") // some group exists → "must select group"

	rec := do(e, http.MethodPost, "/accounts/"+a.ID.String()+"/holdings/by-ticker", url.Values{
		"ticker":   {"UNKNOWN"},
		"quantity": {"1"},
		// no group_id
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "그룹을 선택해야") {
		t.Errorf("expected group selection error, got: %s", rec.Body.String())
	}
}
