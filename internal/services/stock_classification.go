package services

import (
	"context"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// AssetClassifier classifies a security as "etf" or "stock" given its ticker and
// (for overseas-listed securities) its exchange code. It returns "" when the
// classification is unknown. Implemented by a KIS-backed adapter in the container.
type AssetClassifier interface {
	ClassifyAssetClass(ticker, exchange string) (string, error)
}

// assetClassUpdater persists an asset_class onto a stock.
type assetClassUpdater interface {
	UpdateAssetClass(ctx context.Context, id uuidx.UUID, assetClass string) (models.Stock, error)
}

// classifyStock sets asset_class on a stock when it is currently unclassified and
// the classifier yields a recognized value ("etf"/"stock"). It returns the
// possibly-updated stock and whether it changed. A nil classifier or an
// already-classified stock is a no-op. Classification is best-effort: callers
// treat a returned error as non-fatal (the stock keeps its nil asset_class).
func classifyStock(
	ctx context.Context,
	updater assetClassUpdater,
	classifier AssetClassifier,
	st models.Stock,
) (models.Stock, bool, error) {
	if classifier == nil || st.AssetClass != nil {
		return st, false, nil
	}
	exchange := ""
	if st.Exchange != nil {
		exchange = *st.Exchange
	}
	ac, err := classifier.ClassifyAssetClass(st.Ticker, exchange)
	if err != nil {
		return st, false, err
	}
	if ac != "etf" && ac != "stock" {
		return st, false, nil
	}
	updated, err := updater.UpdateAssetClass(ctx, st.ID, ac)
	if err != nil {
		return st, false, err
	}
	return updated, true, nil
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

// ClassifyAll classifies every stock that currently has a nil asset_class.
// Per-stock failures are counted and skipped; the run continues to the end.
func (s *StockClassificationService) ClassifyAll(ctx context.Context) (StockClassificationResult, error) {
	var res StockClassificationResult
	stocks, err := s.stocks.ListAll(ctx)
	if err != nil {
		return res, err
	}
	res.Total = len(stocks)
	for _, st := range stocks {
		if st.AssetClass != nil {
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
