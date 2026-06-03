package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/stockformat"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

const (
	_maxSyncLogBytes  = 10 * 1024 * 1024
	_logBackupCount   = 5
	_defaultSyncGroup = "KIS 자동동기화"
)

// BalanceClient fetches live account balance from an external brokerage (e.g. KIS).
type BalanceClient interface {
	FetchAccountSnapshot(cano, acntPrdtCd string) (models.KisAccountSnapshot, error)
}

// KisEmptySnapshotError guards against silently wiping holdings when KIS returns no positions.
type KisEmptySnapshotError struct{ msg string }

func (e *KisEmptySnapshotError) Error() string { return e.msg }

// IsKisEmptySnapshotError reports whether err is a KisEmptySnapshotError.
func IsKisEmptySnapshotError(err error) bool {
	var target *KisEmptySnapshotError
	return errors.As(err, &target)
}

// Unexported interfaces for repo injection — enable mock testing without a DB.

type syncAccountRepo interface {
	UpdateNameCash(ctx context.Context, id uuidx.UUID, name string, cashBalance numeric.Decimal) (models.Account, error)
}

type syncHoldingRepo interface {
	ListByAccount(ctx context.Context, accountID uuidx.UUID) ([]models.Holding, error)
	Create(ctx context.Context, accountID, stockID uuidx.UUID, quantity numeric.Decimal) (models.Holding, error)
	Update(ctx context.Context, id uuidx.UUID, quantity numeric.Decimal) (models.Holding, error)
	Delete(ctx context.Context, id uuidx.UUID) error
}

type syncStockRepo interface {
	ListAll(ctx context.Context) ([]models.Stock, error)
	Create(ctx context.Context, ticker string, groupID uuidx.UUID) (models.Stock, error)
	UpdateName(ctx context.Context, id uuidx.UUID, name string) (models.Stock, error)
	UpdateAssetClass(ctx context.Context, id uuidx.UUID, assetClass string) (models.Stock, error)
}

type syncGroupRepo interface {
	ListAll(ctx context.Context) ([]models.Group, error)
	Create(ctx context.Context, name string, targetPercentage float64) (models.Group, error)
}

// KisAccountSyncService syncs a local account's holdings and cash with KIS balance data.
type KisAccountSyncService struct {
	accounts         syncAccountRepo
	holdings         syncHoldingRepo
	stocks           syncStockRepo
	groups           syncGroupRepo
	client           BalanceClient
	classifier       AssetClassifier // nil = asset_class classification disabled
	defaultGroupName string
	logPath          string
}

// SetClassifier wires a KIS-backed asset classifier so that stocks encountered
// during sync get their asset_class backfilled when currently unclassified.
// nil disables classification.
func (s *KisAccountSyncService) SetClassifier(c AssetClassifier) {
	s.classifier = c
}

// NewKisAccountSyncService constructs the sync service. logPath="" disables JSONL logging.
func NewKisAccountSyncService(
	accounts syncAccountRepo,
	holdings syncHoldingRepo,
	stocks syncStockRepo,
	groups syncGroupRepo,
	client BalanceClient,
	logPath string,
) *KisAccountSyncService {
	return &KisAccountSyncService{
		accounts:         accounts,
		holdings:         holdings,
		stocks:           stocks,
		groups:           groups,
		client:           client,
		defaultGroupName: _defaultSyncGroup,
		logPath:          logPath,
	}
}

// SyncAccount fetches a KIS balance snapshot and reconciles local holdings + cash.
// Pass allowEmptySnapshot=true only when the account has been fully liquidated.
func (s *KisAccountSyncService) SyncAccount(
	ctx context.Context,
	account models.Account,
	cano, acntPrdtCd string,
	allowEmptySnapshot bool,
) (models.KisAccountSyncResult, error) {
	baseEvent := map[string]any{
		"account_id":   account.ID.String(),
		"cano":         cano,
		"acnt_prdt_cd": acntPrdtCd,
	}

	snapshot, err := s.client.FetchAccountSnapshot(cano, acntPrdtCd)
	if err != nil {
		s.logEvent(baseEvent, map[string]any{
			"event":      "sync_snapshot_error",
			"error_type": fmt.Sprintf("%T", err),
			"error":      err.Error(),
		})
		return models.KisAccountSyncResult{}, err
	}

	existingHoldings, err := s.holdings.ListByAccount(ctx, account.ID)
	if err != nil {
		return models.KisAccountSyncResult{}, err
	}

	if len(snapshot.Holdings) == 0 && len(existingHoldings) > 0 && !allowEmptySnapshot {
		s.logEvent(baseEvent, map[string]any{
			"event":                  "sync_guard_empty_snapshot",
			"existing_holding_count": len(existingHoldings),
		})
		return models.KisAccountSyncResult{}, &KisEmptySnapshotError{
			msg: "KIS 스냅샷이 비어 있어 기존 보유 내역을 보호합니다. 실제로 전량 매도된 경우 allow_empty_snapshot=True로 재실행하세요.",
		}
	}

	allStocks, err := s.stocks.ListAll(ctx)
	if err != nil {
		return models.KisAccountSyncResult{}, err
	}
	stocksByTicker := make(map[string]models.Stock, len(allStocks))
	for _, st := range allStocks {
		stocksByTicker[st.Ticker] = st
	}

	var syncGroupID *uuidx.UUID
	createdStockCount := 0

	// targetByStockID: stock_id → aggregated quantity (snapshot order = ticker-sorted).
	type targetEntry struct {
		stockID  uuidx.UUID
		ticker   string
		quantity numeric.Decimal
	}
	var targetOrder []targetEntry
	targetQty := make(map[uuidx.UUID]numeric.Decimal)

	for _, pos := range snapshot.Holdings {
		st, known := stocksByTicker[pos.Ticker]
		if !known {
			if syncGroupID == nil {
				gid, err := s.getOrCreateSyncGroupID(ctx)
				if err != nil {
					return models.KisAccountSyncResult{}, err
				}
				syncGroupID = &gid
			}
			st, err = s.stocks.Create(ctx, pos.Ticker, *syncGroupID)
			if err != nil {
				return models.KisAccountSyncResult{}, err
			}
			if pos.Name != "" {
				st, err = s.stocks.UpdateName(ctx, st.ID, stockformat.FormatName(pos.Name))
				if err != nil {
					return models.KisAccountSyncResult{}, err
				}
			}
			stocksByTicker[st.Ticker] = st
			createdStockCount++
		} else if st.Name == "" && pos.Name != "" {
			st, err = s.stocks.UpdateName(ctx, st.ID, stockformat.FormatName(pos.Name))
			if err != nil {
				return models.KisAccountSyncResult{}, err
			}
			stocksByTicker[st.Ticker] = st
		}

		// Backfill asset_class for newly-seen or still-unclassified stocks so the
		// rebalance engine can honor tax-account eligibility (e.g. an IRP/연금 may
		// buy a domestic-listed ETF but not an individual stock). Best-effort: a
		// classification failure never blocks the sync.
		if s.classifier != nil && st.AssetClass == nil {
			if updated, changed, cerr := classifyStock(ctx, s.stocks, s.classifier, st); cerr != nil {
				s.logEvent(baseEvent, map[string]any{
					"event":  "sync_classify_error",
					"ticker": st.Ticker,
					"error":  cerr.Error(),
				})
			} else if changed {
				st = updated
				stocksByTicker[st.Ticker] = st
			}
		}

		if _, seen := targetQty[st.ID]; !seen {
			targetOrder = append(targetOrder, targetEntry{stockID: st.ID, ticker: pos.Ticker})
		}
		targetQty[st.ID] = numeric.Wrap(targetQty[st.ID].Add(pos.Quantity.Decimal))
	}
	for i := range targetOrder {
		targetOrder[i].quantity = targetQty[targetOrder[i].stockID]
	}

	existingByStockID := make(map[uuidx.UUID][]models.Holding)
	for _, h := range existingHoldings {
		existingByStockID[h.StockID] = append(existingByStockID[h.StockID], h)
	}

	var holdingChanges []models.HoldingSyncDetail

	// Create / update in ticker-sorted (snapshot) order.
	for _, te := range targetOrder {
		existing := existingByStockID[te.stockID]
		if len(existing) == 0 {
			if _, err := s.holdings.Create(ctx, account.ID, te.stockID, te.quantity); err != nil {
				return models.KisAccountSyncResult{}, err
			}
			newQ := te.quantity
			holdingChanges = append(holdingChanges, models.HoldingSyncDetail{
				Ticker:      te.ticker,
				Action:      "created",
				NewQuantity: &newQ,
			})
			continue
		}
		primary := existing[0]
		if !primary.Quantity.Equal(te.quantity.Decimal) {
			oldQ := primary.Quantity
			newQ := te.quantity
			holdingChanges = append(holdingChanges, models.HoldingSyncDetail{
				Ticker:      te.ticker,
				Action:      "updated",
				OldQuantity: &oldQ,
				NewQuantity: &newQ,
			})
			if _, err := s.holdings.Update(ctx, primary.ID, te.quantity); err != nil {
				return models.KisAccountSyncResult{}, err
			}
		}
		for _, dup := range existing[1:] {
			if err := s.holdings.Delete(ctx, dup.ID); err != nil {
				return models.KisAccountSyncResult{}, err
			}
		}
	}

	// Delete holdings for tickers removed from snapshot. Iterate existingHoldings
	// (not map range) to keep order deterministic.
	seenForDelete := make(map[uuidx.UUID]bool)
	for _, h := range existingHoldings {
		if seenForDelete[h.StockID] {
			continue
		}
		seenForDelete[h.StockID] = true
		if _, inTarget := targetQty[h.StockID]; inTarget {
			continue
		}
		ticker := "?"
		for _, st := range allStocks {
			if st.ID == h.StockID {
				ticker = st.Ticker
				break
			}
		}
		for _, toDelete := range existingByStockID[h.StockID] {
			oldQ := toDelete.Quantity
			holdingChanges = append(holdingChanges, models.HoldingSyncDetail{
				Ticker:      ticker,
				Action:      "deleted",
				OldQuantity: &oldQ,
			})
			if err := s.holdings.Delete(ctx, toDelete.ID); err != nil {
				return models.KisAccountSyncResult{}, err
			}
		}
	}

	oldCash := account.CashBalance
	if _, err := s.accounts.UpdateNameCash(ctx, account.ID, account.Name, snapshot.CashBalance); err != nil {
		return models.KisAccountSyncResult{}, err
	}

	result := models.KisAccountSyncResult{
		AccountID:         account.ID,
		CashBalance:       snapshot.CashBalance,
		OldCashBalance:    oldCash,
		HoldingCount:      len(targetQty),
		CreatedStockCount: createdStockCount,
		HoldingChanges:    holdingChanges,
	}

	changesLog := make([]map[string]any, len(holdingChanges))
	for i, c := range holdingChanges {
		m := map[string]any{"ticker": c.Ticker, "action": c.Action}
		if c.OldQuantity != nil {
			m["old_quantity"] = c.OldQuantity.String()
		}
		if c.NewQuantity != nil {
			m["new_quantity"] = c.NewQuantity.String()
		}
		changesLog[i] = m
	}
	s.logEvent(baseEvent, map[string]any{
		"event":                "sync_success",
		"old_cash_balance":     oldCash.String(),
		"cash_balance":         snapshot.CashBalance.String(),
		"holding_count":        len(targetQty),
		"created_stock_count":  createdStockCount,
		"allow_empty_snapshot": allowEmptySnapshot,
		"holding_changes":      changesLog,
	})

	return result, nil
}

// ValidateAccount confirms KIS credentials by fetching a balance snapshot.
func (s *KisAccountSyncService) ValidateAccount(cano, acntPrdtCd string) error {
	_, err := s.client.FetchAccountSnapshot(cano, acntPrdtCd)
	return err
}

func (s *KisAccountSyncService) getOrCreateSyncGroupID(ctx context.Context) (uuidx.UUID, error) {
	groups, err := s.groups.ListAll(ctx)
	if err != nil {
		return uuidx.UUID{}, err
	}
	for _, g := range groups {
		if g.Name == s.defaultGroupName {
			return g.ID, nil
		}
	}
	created, err := s.groups.Create(ctx, s.defaultGroupName, 0.0)
	if err != nil {
		return uuidx.UUID{}, err
	}
	return created.ID, nil
}

// logEvent appends a JSONL event. Never panics — logging must not break sync.
func (s *KisAccountSyncService) logEvent(base, extra map[string]any) {
	if s.logPath == "" {
		return
	}
	payload := map[string]any{"timestamp": time.Now().Format(time.RFC3339Nano)}
	for k, v := range base {
		payload[k] = v
	}
	for k, v := range extra {
		payload[k] = v
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.rotateIfNeeded()
	f, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // log path is operator-controlled
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString(string(data) + "\n")
}

func (s *KisAccountSyncService) rotateIfNeeded() {
	info, err := os.Stat(s.logPath)
	if err != nil || info.Size() < _maxSyncLogBytes {
		return
	}
	for i := _logBackupCount - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", s.logPath, i)
		dst := fmt.Sprintf("%s.%d", s.logPath, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(s.logPath, s.logPath+".1")
}
