package handlers_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
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

// --- sync ---

// TestSyncAccountNilService checks that POST /accounts/:id/sync returns the
// "KIS 계좌 동기화 서비스가 설정되지 않았습니다" message when AccountSync is nil
// (the default in-memory container).
func TestSyncAccountNilService(t *testing.T) {
	e, c := setupAccounts(t)
	a := seedAccount(t, c, "동기화계좌")

	rec := do(e, http.MethodPost, "/accounts/"+a.ID.String()+"/sync", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "KIS 계좌 동기화 서비스가 설정되지 않았습니다") {
		t.Errorf("nil-service message missing:\n%s", rec.Body.String())
	}
}

// TestSyncAccountInvalidUUID checks that POST /accounts/bad-uuid/sync returns the
// "잘못된 계좌 ID입니다" message.
func TestSyncAccountInvalidUUID(t *testing.T) {
	e, _ := setupAccounts(t)
	rec := do(e, http.MethodPost, "/accounts/bad-uuid/sync", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "잘못된 계좌 ID입니다") {
		t.Errorf("invalid-uuid message missing:\n%s", rec.Body.String())
	}
}

// TestSyncAccountNotFound checks that POST /accounts/{valid-but-nonexistent-uuid}/sync
// returns the "계좌를 찾을 수 없습니다" message.
//
// NOTE: The syncAccount handler checks AccountSync == nil BEFORE querying the DB,
// so AccountSync must be non-nil to reach the not-found branch. We wire a
// KisAccountSyncService with a nil BalanceClient, which is safe because
// SyncAccount (the only call path to the client) is never reached when GetByID
// returns nil.
//
// normalizeKisAccountNo is an unexported function in the handlers package and
// TestSyncAccountNormalizeKisAccountNo exercises normalizeKisAccountNo indirectly:
// account has a kis_account_no set; with AccountSync wired and a nil balance client
// that returns empty snapshot, normalizeKisAccountNo is called to parse the number.
func TestSyncAccountNormalizeKisAccountNo(t *testing.T) {
	e, c := setupAccounts(t)
	ctx := context.Background()

	c.AccountSync = services.NewKisAccountSyncService(
		c.Accounts, c.Holdings, c.Stocks, c.Groups,
		&mockEmptyBalanceClient{},
		"",
	)

	acc, err := c.Accounts.Create(ctx, "KIS 계좌", numeric.Zero)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	updated, err := c.Accounts.Update(ctx, acc.ID, acc.Name, acc.CashBalance,
		sql.NullString{String: "12345678-01", Valid: true},
		sql.NullInt64{})
	if err != nil {
		t.Fatalf("update account: %v", err)
	}

	rec := do(e, http.MethodPost, "/accounts/"+updated.ID.String()+"/sync", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

type mockEmptyBalanceClient struct{}

func (m *mockEmptyBalanceClient) FetchAccountSnapshot(_, _ string) (models.KisAccountSnapshot, error) {
	return models.KisAccountSnapshot{}, nil
}

type trackingBalanceClient struct{ called bool }

func (c *trackingBalanceClient) FetchAccountSnapshot(_, _ string) (models.KisAccountSnapshot, error) {
	c.called = true
	return models.KisAccountSnapshot{}, nil
}

// TestSyncAccountKeyIDRouting verifies that an account with KisAPIKeyID=2 uses the
// key-2 sync service, not the default key-1 service.
func TestSyncAccountKeyIDRouting(t *testing.T) {
	e, c := setupAccounts(t)
	ctx := context.Background()

	key1 := &trackingBalanceClient{}
	c.AccountSync = services.NewKisAccountSyncService(
		c.Accounts, c.Holdings, c.Stocks, c.Groups, key1, "",
	)
	key2 := &trackingBalanceClient{}
	c.AccountSyncByKeyID = map[int64]*services.KisAccountSyncService{
		2: services.NewKisAccountSyncService(c.Accounts, c.Holdings, c.Stocks, c.Groups, key2, ""),
	}

	acc, err := c.Accounts.Create(ctx, "여유금", numeric.Zero)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	updated, err := c.Accounts.Update(ctx, acc.ID, acc.Name, acc.CashBalance,
		sql.NullString{String: "4659285601", Valid: true},
		sql.NullInt64{Int64: 2, Valid: true},
	)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	rec := do(e, http.MethodPost, "/accounts/"+updated.ID.String()+"/sync", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if key1.called {
		t.Error("key-1 client called for KisAPIKeyID=2 account")
	}
	if !key2.called {
		t.Error("key-2 client not called for KisAPIKeyID=2 account")
	}
}

// cannot be tested directly from handlers_test. Its behaviour is implicitly
// covered whenever syncAccount processes a KIS account number; direct unit tests
// would require either exporting it or moving to a white-box test file.
func TestSyncAccountNotFound(t *testing.T) {
	e, c := setupAccounts(t)

	// Wire a non-nil AccountSync so the handler progresses past the nil check.
	c.AccountSync = services.NewKisAccountSyncService(
		c.Accounts, c.Holdings, c.Stocks, c.Groups, nil, "",
	)

	nonexistent := uuidx.New()
	rec := do(e, http.MethodPost, "/accounts/"+nonexistent.String()+"/sync", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "계좌를 찾을 수 없습니다") {
		t.Errorf("not-found message missing:\n%s", rec.Body.String())
	}
}
