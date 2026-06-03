package services

import (
	"context"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

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
	if classifier == nil || (st.AssetClass != nil && st.SecurityGroup != nil) {
		return st, false, nil
	}
	exchange := ""
	if st.Exchange != nil {
		exchange = *st.Exchange
	}
	ac, sg, err := classifier.Classify(st.Ticker, exchange)
	if err != nil {
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
	Classified int // newly assigned an asset_class
	Skipped    int // already classified, or classifier returned no signal
	Failed     int // classifier or persistence errored (non-fatal)
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
}

// NewStockClassificationService constructs the service. A nil classifier yields a
// disabled service (Enabled() == false).
func NewStockClassificationService(stocks classifyStockRepo, classifier AssetClassifier) *StockClassificationService {
	return &StockClassificationService{stocks: stocks, classifier: classifier}
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
	for _, st := range stocks {
		if st.AssetClass != nil && st.SecurityGroup != nil {
			res.Skipped++
			continue
		}
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
