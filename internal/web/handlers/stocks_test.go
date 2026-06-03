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
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

func setupStocks(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewGroupHandler(c).Register(e)
	handlers.NewStockHandler(c).Register(e)
	return e, c
}

func seedGroup(t *testing.T, c *container.Container, name string) models.Group {
	t.Helper()
	g, err := c.Groups.Create(context.Background(), name, 0)
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}
	return g
}

func seedStock(t *testing.T, c *container.Container, ticker string, g models.Group) models.Stock {
	t.Helper()
	s, err := c.Stocks.Create(context.Background(), ticker, g.ID)
	if err != nil {
		t.Fatalf("seed stock: %v", err)
	}
	return s
}

// --- list ---

func TestStocksListOK(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "성장주")
	seedStock(t, c, "AAPL", g)

	rec := do(e, http.MethodGet, "/groups/"+g.ID.String()+"/stocks", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "AAPL") {
		t.Error("AAPL missing from list")
	}
	if !strings.Contains(rec.Body.String(), "성장주 종목") {
		t.Errorf("page title missing: %s", rec.Body.String()[:200])
	}
}

func TestStocksListGroupNotFound(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	// valid uuid but group deleted
	if err := c.Groups.Delete(context.Background(), g.ID); err != nil {
		t.Fatalf("delete group: %v", err)
	}

	rec := do(e, http.MethodGet, "/groups/"+g.ID.String()+"/stocks", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestStocksListBadGroupUUID(t *testing.T) {
	e, _ := setupStocks(t)
	rec := do(e, http.MethodGet, "/groups/not-a-uuid/stocks", nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

// --- create ---

func TestStockCreateRendersRow(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")

	rec := do(e, http.MethodPost, "/groups/"+g.ID.String()+"/stocks", url.Values{
		"ticker": {"aapl"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	// ticker should be uppercased
	if !strings.Contains(rec.Body.String(), "AAPL") {
		t.Errorf("uppercased ticker missing: %s", rec.Body.String())
	}
}

func TestStockCreateTickerAbsent422(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")

	rec := do(e, http.MethodPost, "/groups/"+g.ID.String()+"/stocks", url.Values{})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

// No empty-check on create (parity: Python does not check empty ticker on create).
func TestStockCreateWhitespaceTicker(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")

	rec := do(e, http.MethodPost, "/groups/"+g.ID.String()+"/stocks", url.Values{
		"ticker": {"   "},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no empty-check on create)", rec.Code)
	}
	// should store empty-string ticker
	stocks, _ := c.Stocks.ListByGroup(context.Background(), g.ID)
	if len(stocks) != 1 || stocks[0].Ticker != "" {
		t.Fatalf("expected empty ticker, got %+v", stocks)
	}
}

// --- row ---

func TestStockRowOK(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "MSFT", g)

	rec := do(e, http.MethodGet, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "MSFT") {
		t.Error("ticker missing from row")
	}
}

func TestStockRowWrongGroup404(t *testing.T) {
	e, c := setupStocks(t)
	g1 := seedGroup(t, c, "g1")
	g2 := seedGroup(t, c, "g2")
	s := seedStock(t, c, "X", g1)

	rec := do(e, http.MethodGet, "/groups/"+g2.ID.String()+"/stocks/"+s.ID.String(), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestStockRowBadStockUUID(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")

	rec := do(e, http.MethodGet, "/groups/"+g.ID.String()+"/stocks/bad-uuid", nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

// --- edit form ---

func TestStockEditFormRendersInputs(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "NVDA", g)

	rec := do(e, http.MethodGet, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String()+"/edit", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="NVDA"`) {
		t.Errorf("ticker input missing: %s", body)
	}
	if !strings.Contains(body, `name="target_group_id"`) {
		t.Errorf("group select missing: %s", body)
	}
	if !strings.Contains(body, `hx-put="/groups/`+g.ID.String()+`/stocks/`+s.ID.String()+`"`) {
		t.Errorf("hx-put missing: %s", body)
	}
}

// --- update ---

func TestStockUpdateTickerOK(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "OLD", g)

	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker": {"new"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "NEW") {
		t.Errorf("updated ticker missing: %s", rec.Body.String())
	}
}

func TestStockUpdateMoveGroup(t *testing.T) {
	e, c := setupStocks(t)
	g1 := seedGroup(t, c, "g1")
	g2 := seedGroup(t, c, "g2")
	s := seedStock(t, c, "MOVE", g1)

	rec := do(e, http.MethodPut, "/groups/"+g1.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker":          {"MOVE"},
		"target_group_id": {g2.ID.String()},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	// row buttons must point to new group (updated.group_id)
	body := rec.Body.String()
	if !strings.Contains(body, "/groups/"+g2.ID.String()+"/stocks/") {
		t.Errorf("row still points to old group: %s", body)
	}
}

func TestStockUpdateClearsAssetClass(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "CLR", g)
	if _, err := c.Stocks.UpdateAssetClass(context.Background(), s.ID, "etf"); err != nil {
		t.Fatalf("seed asset class: %v", err)
	}

	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker":      {"CLR"},
		"asset_class": {""},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	got, err := c.Stocks.GetByID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AssetClass != nil {
		t.Fatalf("asset_class after clear = %v, want nil", *got.AssetClass)
	}
}

func TestStockUpdateSetsSecurityGroup(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "SEC", g)

	// lowercase input must be normalized to uppercase KIS code on persist.
	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker":         {"SEC"},
		"security_group": {"ef"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	got, err := c.Stocks.GetByID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SecurityGroup == nil || *got.SecurityGroup != "EF" {
		t.Fatalf("security_group = %v, want EF", got.SecurityGroup)
	}
}

func TestStockUpdateClearsSecurityGroup(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "SEC", g)
	if _, err := c.Stocks.UpdateSecurityGroup(context.Background(), s.ID, "ST"); err != nil {
		t.Fatalf("seed security group: %v", err)
	}

	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker":         {"SEC"},
		"security_group": {""},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	got, err := c.Stocks.GetByID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SecurityGroup != nil {
		t.Fatalf("security_group after clear = %v, want nil", *got.SecurityGroup)
	}
}

func TestStockUpdateEmptyTicker422(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "T", g)

	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker": {"   "},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 for empty ticker on update", rec.Code)
	}
}

func TestStockUpdateTickerAbsent422(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "T", g)

	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestStockUpdateBadTargetGroupUUID422(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "T", g)

	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker":          {"T"},
		"target_group_id": {"not-a-uuid"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestStockUpdateNonExistentTargetGroup404(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "T", g)
	g2 := seedGroup(t, c, "g2")
	// delete g2 so the uuid is valid but group doesn't exist
	if err := c.Groups.Delete(context.Background(), g2.ID); err != nil {
		t.Fatalf("delete group: %v", err)
	}

	rec := do(e, http.MethodPut, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker":          {"T"},
		"target_group_id": {g2.ID.String()},
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestStockUpdateWrongGroup404(t *testing.T) {
	e, c := setupStocks(t)
	g1 := seedGroup(t, c, "g1")
	g2 := seedGroup(t, c, "g2")
	s := seedStock(t, c, "T", g1)

	// PUT to g2's path but stock belongs to g1
	rec := do(e, http.MethodPut, "/groups/"+g2.ID.String()+"/stocks/"+s.ID.String(), url.Values{
		"ticker": {"T"},
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestStockUpdateBadGroupUUID422(t *testing.T) {
	e, _ := setupStocks(t)
	rec := do(e, http.MethodPut, "/groups/bad/stocks/also-bad", url.Values{"ticker": {"T"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

// --- delete ---

func TestStockDeleteOK(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "DEL", g)

	rec := do(e, http.MethodDelete, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	stocks, _ := c.Stocks.ListByGroup(context.Background(), g.ID)
	if len(stocks) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(stocks))
	}
}

func TestStockDeleteBadUUID(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	rec := do(e, http.MethodDelete, "/groups/"+g.ID.String()+"/stocks/bad-uuid", nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestStockEditFormBadUUID(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	rec := do(e, http.MethodGet, "/groups/"+g.ID.String()+"/stocks/bad-uuid/edit", nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

// delete is unconditional: no existence/ownership check (parity).
func TestStockDeleteNonExistentOK(t *testing.T) {
	e, c := setupStocks(t)
	g := seedGroup(t, c, "g")
	s := seedStock(t, c, "T", g)
	if err := c.Stocks.Delete(context.Background(), s.ID); err != nil {
		t.Fatalf("pre-delete: %v", err)
	}

	// delete again — should still be 200
	rec := do(e, http.MethodDelete, "/groups/"+g.ID.String()+"/stocks/"+s.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for unconditional delete", rec.Code)
	}
}
