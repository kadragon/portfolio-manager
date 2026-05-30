package services

import "github.com/kadragon/portfolio-manager/internal/numeric"

// ExchangeRateService provides USD/KRW exchange rates.
// Phase 6 implements the fixed-rate variant; EXIM API client added later.
type ExchangeRateService struct {
	usdKRW numeric.Decimal
}

// NewFixedExchangeRateService creates a service with a fixed USD/KRW rate.
func NewFixedExchangeRateService(usdKRW numeric.Decimal) *ExchangeRateService {
	return &ExchangeRateService{usdKRW: usdKRW}
}

// GetUSDKRW returns the USD/KRW exchange rate.
func (s *ExchangeRateService) GetUSDKRW() numeric.Decimal {
	return s.usdKRW
}
