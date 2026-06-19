package services

import (
	"context"
	"errors"
	"time"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// AssetClassUnknown is the sentinel persisted onto a stock's asset_class when it
// could not be resolved (classifier error or no signal). It is itself terminal
// for the "already classified" skip-gates, so the ticker is not re-queried
// against KIS on every sync/ClassifyAll. It is stamped ONLY on asset_class —
// never on security_group, which keeps its KIS scty_grp_id_cd value space.
// Downstream consumers (e.g. assetIsETF) treat any non-"etf" value as non-ETF,
// so the sentinel is safe. A user can reset asset_class to "미분류" (empty) via
// the edit form to force a re-classification on the next sync.
const AssetClassUnknown = "unknown"

// errClassifyNoSignal marks the case where the classifier returned no usable
// signal (no error, but no asset class) and the stock was sentinel-tagged. It
// lets ClassifyAll count the stock as Failed rather than Classified.
var errClassifyNoSignal = errors.New("no classification signal")

// isUnknown reports whether a nullable column holds the sentinel value.
func isUnknown(s *string) bool {
	return s != nil && *s == AssetClassUnknown
}

// AssetClassifier resolves a security's asset class ("etf"/"stock") and its
// normalized KIS security-group code given its ticker and (for overseas-listed
// securities) its exchange code. Either return value is "" when unknown.
// Implemented by a KIS-backed adapter in the container.
type AssetClassifier interface {
	Classify(ticker, exchange string) (assetClass, securityGroup string, err error)
}

// assetClassUpdater persists an asset_class and/or security_group onto a stock.
type assetClassUpdater interface {
	UpdateAssetClass(ctx context.Context, id uuidx.UUID, assetClass string) (models.Stock, error)
	UpdateSecurityGroup(ctx context.Context, id uuidx.UUID, securityGroup string) (models.Stock, error)
}

// classifyStock backfills asset_class and/or security_group on a stock that is
// missing either, using a single classifier lookup. It returns the
// possibly-updated stock and whether anything changed. A nil classifier or a
// fully-classified stock (both fields set) is a no-op. Classification is
// best-effort: callers treat a returned error as non-fatal.
func classifyStock(
	ctx context.Context,
	updater assetClassUpdater,
	classifier AssetClassifier,
	st models.Stock,
) (models.Stock, bool, error) {
	if classifier == nil || isUnknown(st.AssetClass) || (st.AssetClass != nil && st.SecurityGroup != nil) {
		return st, false, nil
	}
	exchange := ""
	if st.Exchange != nil {
		exchange = *st.Exchange
	}
	ac, sg, err := classifier.Classify(st.Ticker, exchange)
	if err != nil || (ac != "etf" && ac != "stock") {
		if st.AssetClass == nil {
			// Asset class unresolved — stamp the sentinel on asset_class ALONE so
			// the ticker stops being re-queried. security_group keeps its KIS
			// value space (never the sentinel); the asset_class sentinel is
			// itself terminal for the skip-gates. Surface a non-nil error so the
			// run counts this as a failure, not a real classification.
			updated, uerr := updater.UpdateAssetClass(ctx, st.ID, AssetClassUnknown)
			if uerr != nil {
				return st, false, uerr
			}
			if err == nil {
				err = errClassifyNoSignal
			}
			return updated, true, err
		}
		// Asset class already set; classify yielded no new signal. Leave the
		// (cosmetic) missing security_group untouched.
		return st, false, err
	}
	changed := false
	if st.AssetClass == nil && (ac == "etf" || ac == "stock") {
		updated, uerr := updater.UpdateAssetClass(ctx, st.ID, ac)
		if uerr != nil {
			return st, false, uerr
		}
		st = updated
		changed = true
	}
	if st.SecurityGroup == nil && sg != "" {
		updated, uerr := updater.UpdateSecurityGroup(ctx, st.ID, sg)
		if uerr != nil {
			return st, changed, uerr
		}
		st = updated
		changed = true
	}
	return st, changed, nil
}

// StockClassificationResult summarizes a ClassifyAll run.
type StockClassificationResult struct {
	Total      int // stocks inspected
	Classified int // newly assigned a real asset_class ("etf"/"stock")
	Skipped    int // already classified or already sentinel-tagged
	Failed     int // classifier errored or returned no signal; stock sentinel-tagged (non-fatal)
}

type classifyStockRepo interface {
	ListAll(ctx context.Context) ([]models.Stock, error)
	UpdateAssetClass(ctx context.Context, id uuidx.UUID, assetClass string) (models.Stock, error)
	UpdateSecurityGroup(ctx context.Context, id uuidx.UUID, securityGroup string) (models.Stock, error)
}

// StockClassificationService backfills stocks.asset_class via a KIS-backed
// classifier. It is the engine behind the manual "자산구분 분류" action.
type StockClassificationService struct {
	stocks     classifyStockRepo
	classifier AssetClassifier
	// callDelay paces per-stock KIS calls in ClassifyAll to avoid rate-limit /
	// handler timeout. Zero (the constructor default) disables pacing; the
	// container injects a non-zero throttle via SetCallDelay for the live path.
	callDelay time.Duration
}

// NewStockClassificationService constructs the service. A nil classifier yields a
// disabled service (Enabled() == false).
func NewStockClassificationService(stocks classifyStockRepo, classifier AssetClassifier) *StockClassificationService {
	return &StockClassificationService{stocks: stocks, classifier: classifier}
}

// SetCallDelay sets the inter-call pacing for ClassifyAll. Used by the container
// to throttle live KIS traffic; left zero in unit tests for speed.
func (s *StockClassificationService) SetCallDelay(d time.Duration) {
	if s != nil {
		s.callDelay = d
	}
}

// Enabled reports whether a classifier is configured (i.e. KIS is available).
func (s *StockClassificationService) Enabled() bool {
	return s != nil && s.classifier != nil
}

// ClassifyAll classifies every stock that is missing an asset_class or a
// security_group. Per-stock failures are counted and skipped; the run continues
// to the end.
func (s *StockClassificationService) ClassifyAll(ctx context.Context) (StockClassificationResult, error) {
	var res StockClassificationResult
	stocks, err := s.stocks.ListAll(ctx)
	if err != nil {
		return res, err
	}
	res.Total = len(stocks)
	kisCalled := false
	for _, st := range stocks {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if isUnknown(st.AssetClass) || (st.AssetClass != nil && st.SecurityGroup != nil) {
			res.Skipped++
			continue
		}
		// Pace KIS calls — delay BETWEEN attempts only (never before the first,
		// never after the last). time.NewTimer is stopped on cancellation to
		// avoid leaking the timer goroutine.
		if kisCalled && s.callDelay > 0 {
			t := time.NewTimer(s.callDelay)
			select {
			case <-ctx.Done():
				t.Stop()
				return res, ctx.Err()
			case <-t.C:
			}
		}
		kisCalled = true
		_, changed, cerr := classifyStock(ctx, s.stocks, s.classifier, st)
		switch {
		case cerr != nil:
			res.Failed++
		case changed:
			res.Classified++
		default:
			res.Skipped++
		}
	}
	return res, nil
}
