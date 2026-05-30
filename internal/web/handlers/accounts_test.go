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
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

func setupAccounts(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewAccountHandler(c).Register(e)
	return e, c
}

func seedAccount(t *testing.T, c *container.Container, name string) models.Account {
	t.Helper()
	a, err := c.Accounts.Create(context.Background(), name, numeric.Zero)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return a
}

// --- list ---

func TestAccountsListOK(t *testing.T) {
	e, c := setupAccounts(t)
	seedAccount(t, c, "내 계좌")

	rec := do(e, http.MethodGet, "/accounts", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "내 계좌") {
		t.Error("account name missing")
	}
	if !strings.Contains(rec.Body.String(), "₩0") {
		t.Error("cash_balance missing")
	}
}

func TestAccountsListEmpty(t *testing.T) {
	e, _ := setupAccounts(t)
	rec := do(e, http.MethodGet, "/accounts", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "계좌가 없습니다") {
		t.Error("empty message missing")
	}
}

// --- create ---

func TestAccountCreateRendersRow(t *testing.T) {
	e, _ := setupAccounts(t)

	rec := do(e, http.MethodPost, "/accounts", url.Values{
		"name":         {"신규 계좌"},
		"cash_balance": {"1000000"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "신규 계좌") {
		t.Error("name missing")
	}
	if !strings.Contains(body, "₩1,000,000") {
		t.Errorf("formatted cash missing: %s", body)
	}
}

// No empty-name check (parity: Python doesn't validate account name).
func TestAccountCreateEmptyName(t *testing.T) {
	e, _ := setupAccounts(t)
	rec := do(e, http.MethodPost, "/accounts", url.Values{
		"name": {""},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no empty-check on account name)", rec.Code)
	}
}

// --- row ---

func TestAccountRowOK(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "테스트")

	rec := do(e, http.MethodGet, "/accounts/"+a.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "테스트") {
		t.Error("name missing from row")
	}
}

func TestAccountRowNotFound(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "X")
	if err := c.Accounts.DeleteWithHoldings(context.Background(), a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	rec := do(e, http.MethodGet, "/accounts/"+a.ID.String(), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAccountRowBadUUID(t *testing.T) {
	e, _ := setupAccounts(t)
	rec := do(e, http.MethodGet, "/accounts/not-a-uuid", nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

// --- edit form ---

func TestAccountEditFormRendersInputs(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "편집")

	rec := do(e, http.MethodGet, "/accounts/"+a.ID.String()+"/edit", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="name"`) {
		t.Error("name input missing")
	}
	if !strings.Contains(body, `name="cash_balance"`) {
		t.Error("cash_balance input missing")
	}
	if !strings.Contains(body, `name="kis_account_no"`) {
		t.Error("kis_account_no input missing")
	}
	if !strings.Contains(body, `hx-put="/accounts/`+a.ID.String()+`"`) {
		t.Errorf("hx-put missing: %s", body)
	}
}

func TestAccountEditFormNotFound(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "X")
	if err := c.Accounts.DeleteWithHoldings(context.Background(), a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	rec := do(e, http.MethodGet, "/accounts/"+a.ID.String()+"/edit", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- update ---

func TestAccountUpdateOK(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "이전")

	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String(), url.Values{
		"name":         {"이후"},
		"cash_balance": {"500000"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "이후") {
		t.Error("updated name missing")
	}
	if !strings.Contains(rec.Body.String(), "₩500,000") {
		t.Errorf("formatted cash missing: %s", rec.Body.String())
	}
}

func TestAccountUpdateEmptyNameOK(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "이름있음")

	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String(), url.Values{
		"name": {""},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no empty-name validation)", rec.Code)
	}
}

func TestAccountUpdateKisAccountNoSetAndClear(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "KIS")

	// Set KIS account number
	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String(), url.Values{
		"name":           {"KIS"},
		"cash_balance":   {"0"},
		"kis_account_no": {"12345678-01"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("set KIS: status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "KIS 동기화") {
		t.Error("KIS sync button missing when kis_account_no set")
	}

	// Clear KIS account number (empty → NULL)
	rec = do(e, http.MethodPut, "/accounts/"+a.ID.String(), url.Values{
		"name":           {"KIS"},
		"cash_balance":   {"0"},
		"kis_account_no": {""},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("clear KIS: status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "KIS 동기화") {
		t.Error("KIS sync button should not render when kis_account_no cleared")
	}
}

func TestAccountUpdateNotFound(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "X")
	if err := c.Accounts.DeleteWithHoldings(context.Background(), a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	rec := do(e, http.MethodPut, "/accounts/"+a.ID.String(), url.Values{"name": {"X"}})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAccountUpdateBadUUID(t *testing.T) {
	e, _ := setupAccounts(t)
	rec := do(e, http.MethodPut, "/accounts/bad-id", url.Values{"name": {"X"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

// --- delete ---

func TestAccountDeleteOK(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "삭제")

	rec := do(e, http.MethodDelete, "/accounts/"+a.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	accounts, _ := c.Accounts.ListAll(context.Background())
	if len(accounts) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(accounts))
	}
}

// --- bulk-cash ---

func TestBulkCashOK(t *testing.T) {
	e, c := setupAccounts(t)
	a1 := seedAccount(t, c, "A")
	a2 := seedAccount(t, c, "B")

	rec := do(e, http.MethodPut, "/accounts/bulk-cash", url.Values{
		"cash_" + a1.ID.String(): {"1000"},
		"cash_" + a2.ID.String(): {"2000"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("HX-Refresh") != "true" {
		t.Error("HX-Refresh header missing")
	}
}

func TestBulkCashEmptyField422(t *testing.T) {
	e, c := setupAccounts(t)
	seedAccount(t, c, "MyAccount")

	rec := do(e, http.MethodPut, "/accounts/bulk-cash", url.Values{})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "MyAccount") {
		t.Errorf("account name missing in error: %s", rec.Body.String())
	}
}

func TestBulkCashInvalidDecimal422(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "BadInput")

	rec := do(e, http.MethodPut, "/accounts/bulk-cash", url.Values{
		"cash_" + a.ID.String(): {"not-a-number"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "BadInput") {
		t.Errorf("account name missing in error: %s", rec.Body.String())
	}
}

func TestBulkCashPreservesKIS(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "KIS보존")

	// Set KIS fields via full update
	do(e, http.MethodPut, "/accounts/"+a.ID.String(), url.Values{
		"name":           {"KIS보존"},
		"cash_balance":   {"0"},
		"kis_account_no": {"87654321-01"},
	})

	// bulk-cash should not change KIS fields
	rec := do(e, http.MethodPut, "/accounts/bulk-cash", url.Values{
		"cash_" + a.ID.String(): {"9999"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	updated, _ := c.Accounts.GetByID(context.Background(), a.ID)
	if updated == nil || updated.KisAccountNo == nil || *updated.KisAccountNo != "87654321-01" {
		t.Fatalf("KIS preserved: %+v", updated)
	}
}
