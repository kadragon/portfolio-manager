package services

import (
	"context"
	"math"

	"github.com/kadragon/portfolio-manager/internal/models"
)

const defaultOverseasExchange = "NASD"

// OrderClient places a single order via KIS.
type OrderClient interface {
	PlaceOrder(ticker, side string, quantity int, exchange string) (map[string]any, error)
}

// ExecutionRepo persists order execution records.
type ExecutionRepo interface {
	Create(
		ctx context.Context,
		ticker, side string,
		quantity int,
		currency, status, message string,
		exchange string,
		rawResponse map[string]any,
	) (models.OrderExecutionRecord, error)
}

// SyncService syncs account state after order execution.
type SyncService interface {
	SyncAccount() error
}

// RebalanceExecutionService converts recommendations into executable order intents.
type RebalanceExecutionService struct {
	orderClient OrderClient
	execRepo    ExecutionRepo
	syncService SyncService
}

// NewRebalanceExecutionService builds the service. All deps are optional.
func NewRebalanceExecutionService(
	orderClient OrderClient,
	execRepo ExecutionRepo,
	syncService SyncService,
) *RebalanceExecutionService {
	return &RebalanceExecutionService{
		orderClient: orderClient,
		execRepo:    execRepo,
		syncService: syncService,
	}
}

// OrderIntentResult holds executable intents and skipped (qty=0) items.
type OrderIntentResult struct {
	Intents []models.OrderIntent
	Skipped []models.OrderIntent
}

// CreateOrderIntents converts recommendations to executable intents.
// exchangeMap is an optional ticker→exchange override (nil = use defaults).
// Sells are ordered before buys; qty is floored; qty=0 → skipped.
func (s *RebalanceExecutionService) CreateOrderIntents(
	recs []models.RebalanceRecommendation,
	exchangeMap map[string]string,
) OrderIntentResult {
	var intents []models.OrderIntent
	var skipped []models.OrderIntent

	for _, rec := range recs {
		var qty int
		if rec.Quantity != nil {
			qty = int(math.Floor(rec.Quantity.InexactFloat64()))
		}

		exchange := ""
		if !isDomesticTicker(rec.Ticker) {
			if exchangeMap != nil {
				if ex, ok := exchangeMap[rec.Ticker]; ok && ex != "" {
					exchange = ex
				} else {
					exchange = defaultOverseasExchange
				}
			} else {
				exchange = defaultOverseasExchange
			}
		}

		currency := rec.Currency
		if currency == "" {
			if isDomesticTicker(rec.Ticker) {
				currency = "KRW"
			} else {
				currency = "USD"
			}
		}

		intent := models.OrderIntent{
			Ticker:    rec.Ticker,
			Side:      string(rec.Action),
			Quantity:  qty,
			Currency:  currency,
			Exchange:  exchange,
			StockName: rec.StockName,
		}
		if qty == 0 {
			skipped = append(skipped, intent)
		} else {
			intents = append(intents, intent)
		}
	}

	// sells before buys (stable sort preserves original order within each group)
	stable := make([]models.OrderIntent, 0, len(intents))
	for _, in := range intents {
		if in.Side == "sell" {
			stable = append(stable, in)
		}
	}
	for _, in := range intents {
		if in.Side == "buy" {
			stable = append(stable, in)
		}
	}
	return OrderIntentResult{Intents: stable, Skipped: skipped}
}

// ExecuteRebalanceOrders executes orders and persists results.
// If dryRun=true, no API calls are made and only intents are returned.
func (s *RebalanceExecutionService) ExecuteRebalanceOrders(
	recs []models.RebalanceRecommendation,
	dryRun bool,
	exchangeMap map[string]string,
) models.RebalanceExecutionResult {
	intentResult := s.CreateOrderIntents(recs, exchangeMap)

	var executions []models.OrderExecutionResult
	if !dryRun && s.orderClient != nil {
		for _, intent := range intentResult.Intents {
			resp, err := s.orderClient.PlaceOrder(intent.Ticker, intent.Side, intent.Quantity, intent.Exchange)
			status := "success"
			msg := ""
			if err != nil {
				status = "failed"
				msg = err.Error()
			} else if resp != nil {
				if rtCd, ok := resp["rt_cd"].(string); ok && rtCd != "0" {
					status = "failed"
				}
				if m, ok := resp["msg1"].(string); ok {
					msg = m
				}
			}
			executions = append(executions, models.OrderExecutionResult{
				Intent:      intent,
				Status:      status,
				Message:     msg,
				RawResponse: resp,
			})
		}
	}

	if !dryRun && s.execRepo != nil {
		for _, ex := range executions {
			_, _ = s.execRepo.Create(
				context.Background(),
				ex.Intent.Ticker, ex.Intent.Side,
				ex.Intent.Quantity,
				ex.Intent.Currency, ex.Status, ex.Message,
				ex.Intent.Exchange,
				ex.RawResponse,
			)
		}
		for _, sk := range intentResult.Skipped {
			_, _ = s.execRepo.Create(
				context.Background(),
				sk.Ticker, sk.Side,
				sk.Quantity,
				sk.Currency, "skipped", "",
				sk.Exchange,
				nil,
			)
		}
	}

	syncWarning := ""
	if !dryRun && s.syncService != nil {
		if err := s.syncService.SyncAccount(); err != nil {
			syncWarning = err.Error()
		}
	}

	return models.RebalanceExecutionResult{
		Intents:     intentResult.Intents,
		Skipped:     intentResult.Skipped,
		Executions:  executions,
		SyncWarning: syncWarning,
	}
}
