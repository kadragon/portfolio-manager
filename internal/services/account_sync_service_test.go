package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// --- mock balance client ---

type mockBalanceClient struct {
	snapshot models.KisAccountSnapshot
	err      error
	calls    int
}

func (m *mockBalanceClient) FetchAccountSnapshot(cano, acntPrdtCd string) (models.KisAccountSnapshot, error) {
	m.calls++
	return m.snapshot, m.err
}

// --- mock repos ---

type mockSyncAccountRepo struct {
	updated []mockAccountUpdate
}
type mockAccountUpdate struct {
	id   uuidx.UUID
	name string
	cash numeric.Decimal
}

func (r *mockSyncAccountRepo) UpdateNameCash(_ context.Context, id uuidx.UUID, name string, cash numeric.Decimal) (models.Account, error) {
	r.updated = append(r.updated, mockAccountUpdate{id, name, cash})
	return models.Account{ID: id, Name: name, CashBalance: cash}, nil
}

type mockSyncHoldingRepo struct {
	existing []models.Holding
	created  []mockHoldingCreate
	updated  []mockHoldingUpdate
	deleted  []uuidx.UUID
}
type mockHoldingCreate struct {
	accountID, stockID uuidx.UUID
	quantity           numeric.Decimal
}
type mockHoldingUpdate struct {
	id       uuidx.UUID
	quantity numeric.Decimal
}

func (r *mockSyncHoldingRepo) ListByAccount(_ context.Context, _ uuidx.UUID) ([]models.Holding, error) {
	return r.existing, nil
}
func (r *mockSyncHoldingRepo) Create(_ context.Context, accountID, stockID uuidx.UUID, qty numeric.Decimal) (models.Holding, error) {
	r.created = append(r.created, mockHoldingCreate{accountID, stockID, qty})
	return models.Holding{ID: newTestUUID(), AccountID: accountID, StockID: stockID, Quantity: qty}, nil
}
func (r *mockSyncHoldingRepo) Update(_ context.Context, id uuidx.UUID, qty numeric.Decimal) (models.Holding, error) {
	r.updated = append(r.updated, mockHoldingUpdate{id, qty})
	return models.Holding{ID: id, Quantity: qty}, nil
}
func (r *mockSyncHoldingRepo) Delete(_ context.Context, id uuidx.UUID) error {
	r.deleted = append(r.deleted, id)
	return nil
}

type mockSyncStockRepo struct {
	all     []models.Stock
	created []models.Stock
	named   []mockStockNameUpdate
}
type mockStockNameUpdate struct {
	id   uuidx.UUID
	name string
}

func (r *mockSyncStockRepo) ListAll(_ context.Context) ([]models.Stock, error) {
	return r.all, nil
}
func (r *mockSyncStockRepo) Create(_ context.Context, ticker string, groupID uuidx.UUID) (models.Stock, error) {
	st := models.Stock{ID: newTestUUID(), Ticker: ticker, GroupID: groupID}
	r.all = append(r.all, st)
	r.created = append(r.created, st)
	return st, nil
}
func (r *mockSyncStockRepo) UpdateName(_ context.Context, id uuidx.UUID, name string) (models.Stock, error) {
	r.named = append(r.named, mockStockNameUpdate{id, name})
	for i, st := range r.all {
		if st.ID == id {
			r.all[i].Name = name
			return r.all[i], nil
		}
	}
	return models.Stock{}, errors.New("stock not found")
}

type mockSyncGroupRepo struct {
	all     []models.Group
	created []models.Group
}

func (r *mockSyncGroupRepo) ListAll(_ context.Context) ([]models.Group, error) {
	return r.all, nil
}
func (r *mockSyncGroupRepo) Create(_ context.Context, name string, _ float64) (models.Group, error) {
	g := models.Group{ID: newTestUUID(), Name: name}
	r.all = append(r.all, g)
	r.created = append(r.created, g)
	return g, nil
}

// --- helpers ---

func newTestUUID() uuidx.UUID {
	return uuidx.New()
}

func mustDecimal(s string) numeric.Decimal {
	d, err := numeric.FromString(s)
	if err != nil {
		panic(s)
	}
	return d
}

func makeTestAccount() models.Account {
	return models.Account{
		ID:          newTestUUID(),
		Name:        "테스트 계좌",
		CashBalance: mustDecimal("1000000"),
	}
}

func makeSyncSvc(
	bc *mockBalanceClient,
	accts *mockSyncAccountRepo,
	hlds *mockSyncHoldingRepo,
	stocks *mockSyncStockRepo,
	groups *mockSyncGroupRepo,
) *services.KisAccountSyncService {
	return services.NewKisAccountSyncService(accts, hlds, stocks, groups, bc, "")
}

// --- tests ---

func TestSyncAccount_EmptySnapshotGuard(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()
	holdingID := newTestUUID()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{}} // empty
	hlds := &mockSyncHoldingRepo{
		existing: []models.Holding{{ID: holdingID, AccountID: account.ID, StockID: stockID, Quantity: mustDecimal("10")}},
	}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, &mockSyncStockRepo{}, &mockSyncGroupRepo{})

	_, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err == nil {
		t.Fatal("want KisEmptySnapshotError, got nil")
	}
	if !services.IsKisEmptySnapshotError(err) {
		t.Errorf("want KisEmptySnapshotError, got %T: %v", err, err)
	}
}

func TestSyncAccount_EmptySnapshotAllowed(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()
	holdingID := newTestUUID()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{}}
	hlds := &mockSyncHoldingRepo{
		existing: []models.Holding{{ID: holdingID, AccountID: account.ID, StockID: stockID, Quantity: mustDecimal("5")}},
	}
	stocks := &mockSyncStockRepo{all: []models.Stock{{ID: stockID, Ticker: "005930"}}}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, &mockSyncGroupRepo{})

	result, err := svc.SyncAccount(context.Background(), account, "12345678", "01", true)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	if len(hlds.deleted) != 1 || hlds.deleted[0] != holdingID {
		t.Errorf("want holding deleted, deleted=%v", hlds.deleted)
	}
	if len(result.HoldingChanges) != 1 || result.HoldingChanges[0].Action != "deleted" {
		t.Errorf("want deleted change, got %v", result.HoldingChanges)
	}
	if result.HoldingCount != 0 {
		t.Errorf("want HoldingCount=0, got %d", result.HoldingCount)
	}
}

func TestSyncAccount_CreateHolding(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{
		Holdings: []models.KisHoldingPosition{{Ticker: "005930", Quantity: mustDecimal("10")}},
	}}
	stocks := &mockSyncStockRepo{all: []models.Stock{{ID: stockID, Ticker: "005930"}}}
	hlds := &mockSyncHoldingRepo{}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, &mockSyncGroupRepo{})

	result, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hlds.created) != 1 {
		t.Fatalf("want 1 holding created, got %d", len(hlds.created))
	}
	if !hlds.created[0].quantity.Equal(mustDecimal("10").Decimal) {
		t.Errorf("created qty: want 10, got %v", hlds.created[0].quantity)
	}
	if len(result.HoldingChanges) != 1 || result.HoldingChanges[0].Action != "created" {
		t.Errorf("want created change, got %v", result.HoldingChanges)
	}
	if result.HoldingCount != 1 {
		t.Errorf("want HoldingCount=1, got %d", result.HoldingCount)
	}
}

func TestSyncAccount_UpdateHolding(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()
	holdingID := newTestUUID()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{
		Holdings: []models.KisHoldingPosition{{Ticker: "005930", Quantity: mustDecimal("15")}},
	}}
	stocks := &mockSyncStockRepo{all: []models.Stock{{ID: stockID, Ticker: "005930"}}}
	hlds := &mockSyncHoldingRepo{
		existing: []models.Holding{{ID: holdingID, AccountID: account.ID, StockID: stockID, Quantity: mustDecimal("10")}},
	}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, &mockSyncGroupRepo{})

	_, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hlds.updated) != 1 {
		t.Fatalf("want 1 holding updated, got %d", len(hlds.updated))
	}
	if !hlds.updated[0].quantity.Equal(mustDecimal("15").Decimal) {
		t.Errorf("updated qty: want 15, got %v", hlds.updated[0].quantity)
	}
}

func TestSyncAccount_NoChangeHolding(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()
	holdingID := newTestUUID()

	// Different decimal representations of the same value — must not trigger update.
	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{
		Holdings: []models.KisHoldingPosition{{Ticker: "005930", Quantity: mustDecimal("10.0")}},
	}}
	stocks := &mockSyncStockRepo{all: []models.Stock{{ID: stockID, Ticker: "005930"}}}
	hlds := &mockSyncHoldingRepo{
		existing: []models.Holding{{ID: holdingID, AccountID: account.ID, StockID: stockID, Quantity: numeric.FromInt(10)}},
	}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, &mockSyncGroupRepo{})

	result, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hlds.updated) != 0 {
		t.Errorf("want 0 updates (no-change), got %d updates", len(hlds.updated))
	}
	if len(result.HoldingChanges) != 0 {
		t.Errorf("want 0 changes, got %v", result.HoldingChanges)
	}
}

func TestSyncAccount_DeleteHolding(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()
	holdingID := newTestUUID()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{}} // empty — no holdings
	stocks := &mockSyncStockRepo{all: []models.Stock{{ID: stockID, Ticker: "005930"}}}
	hlds := &mockSyncHoldingRepo{
		existing: []models.Holding{{ID: holdingID, AccountID: account.ID, StockID: stockID, Quantity: mustDecimal("5")}},
	}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, &mockSyncGroupRepo{})

	result, err := svc.SyncAccount(context.Background(), account, "12345678", "01", true) // allow_empty
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hlds.deleted) != 1 || hlds.deleted[0] != holdingID {
		t.Errorf("want holding deleted, deleted=%v", hlds.deleted)
	}
	if len(result.HoldingChanges) != 1 || result.HoldingChanges[0].Action != "deleted" {
		t.Errorf("want deleted change")
	}
}

func TestSyncAccount_CreateNewStock(t *testing.T) {
	account := makeTestAccount()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{
		Holdings: []models.KisHoldingPosition{{Ticker: "AAPL", Quantity: mustDecimal("3"), Name: "Apple Inc."}},
	}}
	stocks := &mockSyncStockRepo{all: []models.Stock{}} // ticker unknown
	hlds := &mockSyncHoldingRepo{}
	groups := &mockSyncGroupRepo{}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, groups)

	result, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stocks.created) != 1 || stocks.created[0].Ticker != "AAPL" {
		t.Errorf("want AAPL stock created, got %v", stocks.created)
	}
	if len(groups.created) != 1 {
		t.Errorf("want sync group created, got %d groups", len(groups.created))
	}
	if groups.created[0].Name != "KIS 자동동기화" {
		t.Errorf("want default group name, got %q", groups.created[0].Name)
	}
	if len(hlds.created) != 1 {
		t.Errorf("want 1 holding created, got %d", len(hlds.created))
	}
	if result.CreatedStockCount != 1 {
		t.Errorf("want CreatedStockCount=1, got %d", result.CreatedStockCount)
	}
}

func TestSyncAccount_UpdateStockName(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{
		Holdings: []models.KisHoldingPosition{{Ticker: "005930", Quantity: mustDecimal("1"), Name: "삼성전자"}},
	}}
	stocks := &mockSyncStockRepo{all: []models.Stock{{ID: stockID, Ticker: "005930", Name: ""}}} // no name
	hlds := &mockSyncHoldingRepo{}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, &mockSyncGroupRepo{})

	_, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stocks.named) != 1 {
		t.Errorf("want stock name updated, got %d updates", len(stocks.named))
	}
	if stocks.named[0].name != "삼성전자" {
		t.Errorf("want name '삼성전자', got %q", stocks.named[0].name)
	}
}

func TestSyncAccount_DeduplicateHoldings(t *testing.T) {
	account := makeTestAccount()
	stockID := newTestUUID()
	h1ID := newTestUUID()
	h2ID := newTestUUID()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{
		Holdings: []models.KisHoldingPosition{{Ticker: "005930", Quantity: mustDecimal("8")}},
	}}
	stocks := &mockSyncStockRepo{all: []models.Stock{{ID: stockID, Ticker: "005930"}}}
	hlds := &mockSyncHoldingRepo{
		existing: []models.Holding{
			{ID: h1ID, AccountID: account.ID, StockID: stockID, Quantity: mustDecimal("5")},
			{ID: h2ID, AccountID: account.ID, StockID: stockID, Quantity: mustDecimal("3")},
		},
	}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, hlds, stocks, &mockSyncGroupRepo{})

	_, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// h1 should be updated to 8; h2 should be deleted
	if len(hlds.updated) != 1 || hlds.updated[0].id != h1ID {
		t.Errorf("want h1 updated, updated=%v", hlds.updated)
	}
	if len(hlds.deleted) != 1 || hlds.deleted[0] != h2ID {
		t.Errorf("want h2 deleted, deleted=%v", hlds.deleted)
	}
}

func TestSyncAccount_UpdatesCashBalance(t *testing.T) {
	account := makeTestAccount()

	bc := &mockBalanceClient{snapshot: models.KisAccountSnapshot{
		CashBalance: mustDecimal("2000000"),
	}}
	accts := &mockSyncAccountRepo{}
	hlds := &mockSyncHoldingRepo{}
	svc := makeSyncSvc(bc, accts, hlds, &mockSyncStockRepo{}, &mockSyncGroupRepo{})

	result, err := svc.SyncAccount(context.Background(), account, "12345678", "01", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accts.updated) != 1 {
		t.Fatalf("want 1 account update, got %d", len(accts.updated))
	}
	if !accts.updated[0].cash.Equal(mustDecimal("2000000").Decimal) {
		t.Errorf("want cash=2000000, got %v", accts.updated[0].cash)
	}
	if !result.CashBalance.Equal(mustDecimal("2000000").Decimal) {
		t.Errorf("result.CashBalance: want 2000000, got %v", result.CashBalance)
	}
	if !result.OldCashBalance.Equal(mustDecimal("1000000").Decimal) {
		t.Errorf("result.OldCashBalance: want 1000000, got %v", result.OldCashBalance)
	}
}

func TestSyncAccount_SnapshotErrorPropagated(t *testing.T) {
	account := makeTestAccount()
	bc := &mockBalanceClient{err: errors.New("KIS API down")}
	svc := makeSyncSvc(bc, &mockSyncAccountRepo{}, &mockSyncHoldingRepo{}, &mockSyncStockRepo{}, &mockSyncGroupRepo{})

	_, err := svc.SyncAccount(context.Background(), account, "12345678", "01", false)
	if err == nil || err.Error() != "KIS API down" {
		t.Errorf("want 'KIS API down', got %v", err)
	}
}
