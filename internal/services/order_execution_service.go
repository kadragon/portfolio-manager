package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/models"
)

// KISOrderClient places a single market order through KIS (domestic or overseas).
type KISOrderClient interface {
	PlaceOrder(ticker, side string, quantity int, exchange string) (map[string]any, error)
}

// TossOrderClient places a single market order through Toss.
type TossOrderClient interface {
	PlaceOrder(accountSeq string, intent models.OrderIntent) (map[string]any, error)
}

// KISOrderClientFactory builds a KISOrderClient scoped to one account's CANO,
// since KIS bakes the account number into the client rather than taking it
// per-call. keyID selects which KIS API key set to authenticate with (nil or
// unknown falls back to key 1).
type KISOrderClientFactory func(keyID *int64, cano, acntPrdtCd string) (KISOrderClient, error)

type orderAccountLister interface {
	ListAll(ctx context.Context) ([]models.Account, error)
}

type orderExecutionRecorder interface {
	Create(
		ctx context.Context,
		ticker, side string,
		quantity int,
		currency, status, message string,
		exchange string,
		rawResponse map[string]any,
	) (models.OrderExecutionRecord, error)
}

// OrderExecutionService places a rebalance order for a named account, routing
// to KIS or Toss depending on how the account is linked, and persists the
// outcome (success or failure) as an audit trail via OrderExecutionRepository.
//
// This re-wires the KIS/Toss order clients and OrderExecutionRepository that
// ADR-0001 deliberately left unreferenced after removing in-app automated
// execution — this service is the "future execution path" it anticipated,
// invoked from the execute-rebalance-plan skill rather than a web handler.
type OrderExecutionService struct {
	accounts          orderAccountLister
	executions        orderExecutionRecorder
	kisFactory        KISOrderClientFactory
	toss              TossOrderClient
	defaultCano       string
	defaultAcntPrdtCd string
}

// NewOrderExecutionService builds the service. toss may be nil if Toss isn't
// configured; defaultCano/defaultAcntPrdtCd are the env-configured primary
// account, used as a fallback for KIS-linked accounts with no KisAccountNo.
func NewOrderExecutionService(
	accounts orderAccountLister,
	executions orderExecutionRecorder,
	kisFactory KISOrderClientFactory,
	toss TossOrderClient,
	defaultCano, defaultAcntPrdtCd string,
) *OrderExecutionService {
	return &OrderExecutionService{
		accounts:          accounts,
		executions:        executions,
		kisFactory:        kisFactory,
		toss:              toss,
		defaultCano:       defaultCano,
		defaultAcntPrdtCd: defaultAcntPrdtCd,
	}
}

// PlaceOrder finds the account by name, routes the order to its linked broker,
// and always persists an OrderExecutionRecord — even on failure — so the
// attempt is auditable. The returned error is non-nil only for problems that
// prevented an order attempt (unknown account, no broker link, bad input);
// a broker-side rejection is reported via record.Status == "failed" instead.
func (s *OrderExecutionService) PlaceOrder(
	ctx context.Context,
	accountName, ticker, side string,
	quantity int,
	exchange, currency string,
) (models.OrderExecutionRecord, error) {
	side = strings.ToLower(strings.TrimSpace(side))
	if side != "buy" && side != "sell" {
		return models.OrderExecutionRecord{}, fmt.Errorf("side must be buy or sell, got %q", side)
	}
	if quantity <= 0 {
		return models.OrderExecutionRecord{}, fmt.Errorf("quantity must be positive, got %d", quantity)
	}

	account, err := s.findAccount(ctx, accountName)
	if err != nil {
		return models.OrderExecutionRecord{}, err
	}

	var raw map[string]any
	var placeErr error

	switch {
	case account.TossAccountSeq != nil:
		if s.toss == nil {
			return models.OrderExecutionRecord{}, fmt.Errorf("account %q is Toss-linked but no Toss client is configured", account.Name)
		}
		seq := strconv.FormatInt(*account.TossAccountSeq, 10)
		raw, placeErr = s.toss.PlaceOrder(seq, models.OrderIntent{
			Ticker: ticker, Side: side, Quantity: quantity, Currency: currency,
			AccountID: account.ID, AccountName: account.Name,
		})

	case account.KisAccountNo != nil && strings.TrimSpace(*account.KisAccountNo) != "":
		cano, acntPrdtCd, nerr := normalizeKisAccountNo(*account.KisAccountNo)
		if nerr != nil {
			return models.OrderExecutionRecord{}, fmt.Errorf("account %q: %w", account.Name, nerr)
		}
		client, ferr := s.kisFactory(account.KisAPIKeyID, cano, acntPrdtCd)
		if ferr != nil {
			return models.OrderExecutionRecord{}, ferr
		}
		raw, placeErr = client.PlaceOrder(ticker, side, quantity, exchange)

	case s.defaultCano != "":
		client, ferr := s.kisFactory(account.KisAPIKeyID, s.defaultCano, s.defaultAcntPrdtCd)
		if ferr != nil {
			return models.OrderExecutionRecord{}, ferr
		}
		raw, placeErr = client.PlaceOrder(ticker, side, quantity, exchange)

	default:
		return models.OrderExecutionRecord{}, fmt.Errorf("account %q is not linked to KIS or Toss", account.Name)
	}

	status, message := classifyOrderResult(raw, placeErr)
	record, cerr := s.executions.Create(ctx, ticker, side, quantity, currency, status, message, exchange, raw)
	if cerr != nil {
		return models.OrderExecutionRecord{}, fmt.Errorf("order attempted (status=%s message=%s) but failed to persist the execution record: %w", status, message, cerr)
	}
	return record, nil
}

// findAccount resolves an account by exact (case-insensitive) name match, or
// falls back to a unique case-insensitive substring match — erroring if the
// name matches zero or more than one account.
func (s *OrderExecutionService) findAccount(ctx context.Context, name string) (models.Account, error) {
	accounts, err := s.accounts.ListAll(ctx)
	if err != nil {
		return models.Account{}, err
	}

	name = strings.TrimSpace(name)
	for _, a := range accounts {
		if strings.EqualFold(a.Name, name) {
			return a, nil
		}
	}

	lower := strings.ToLower(name)
	var matches []models.Account
	for _, a := range accounts {
		if strings.Contains(strings.ToLower(a.Name), lower) {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return models.Account{}, fmt.Errorf("no account found matching %q", name)
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return models.Account{}, fmt.Errorf("account name %q is ambiguous, matches: %s", name, strings.Join(names, ", "))
	}
}

// classifyOrderResult reports the outcome of an order attempt as a persistable
// status. A transport/client error is "failed". Otherwise KIS's rt_cd ("0" =
// success) determines the outcome; Toss's client already errors on a missing
// orderId, so any non-error Toss response is a success.
func classifyOrderResult(raw map[string]any, err error) (status, message string) {
	if err != nil {
		return "failed", err.Error()
	}
	if rtCode, ok := raw["rt_cd"].(string); ok && rtCode != "0" {
		msg, _ := raw["msg1"].(string)
		return "failed", msg
	}
	return "success", ""
}

// normalizeKisAccountNo extracts the 8-digit CANO and 2-digit account-product
// code from a KIS account number string (e.g. "12345678-01" or "1234567801").
// Duplicated from internal/web/handlers.normalizeKisAccountNo (unexported
// there) rather than shared, matching this codebase's existing convention of
// small local re-derivations (see container.go's loadKISAccount).
func normalizeKisAccountNo(s string) (cano, acntPrdtCd string, err error) {
	var digits strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}
	d := digits.String()
	if len(d) != 10 {
		return "", "", fmt.Errorf("invalid KIS account number format (need 8+2 digits): %q", s)
	}
	return d[:8], d[8:], nil
}
