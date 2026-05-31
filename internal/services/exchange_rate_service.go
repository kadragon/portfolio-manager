package services

import (
	"fmt"
	"sync"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

// EximRateClient fetches the USD/KRW rate from the Korea EXIM Bank API.
type EximRateClient interface {
	FetchUSDRate(searchDate string) (float64, error)
}

// ExchangeRateService provides USD/KRW exchange rates.
// Supports a fixed rate or a live EXIM client with 7-day backoff and in-memory cache.
type ExchangeRateService struct {
	mu          sync.Mutex
	fixedRate   *numeric.Decimal
	eximClient  EximRateClient
	cachedRates map[string]numeric.Decimal
}

// NewFixedExchangeRateService creates a service with a fixed USD/KRW rate.
func NewFixedExchangeRateService(usdKRW numeric.Decimal) *ExchangeRateService {
	return &ExchangeRateService{fixedRate: &usdKRW}
}

// NewEximExchangeRateService creates a service that fetches live USD/KRW rates from EXIM.
func NewEximExchangeRateService(client EximRateClient) *ExchangeRateService {
	return &ExchangeRateService{
		eximClient:  client,
		cachedRates: make(map[string]numeric.Decimal),
	}
}

// GetUSDKRW returns the USD/KRW rate.
// Fixed-rate services return immediately; EXIM services try today then up to 6 prior days.
func (s *ExchangeRateService) GetUSDKRW() numeric.Decimal {
	if s.fixedRate != nil {
		return *s.fixedRate
	}
	if s.eximClient == nil {
		return numeric.Zero
	}

	today := datex.FromTime(ktime.NowKST())
	for offset := 0; offset < 7; offset++ {
		candidate := today.Time.AddDate(0, 0, -offset).Format("20060102")
		s.mu.Lock()
		if v, ok := s.cachedRates[candidate]; ok {
			s.mu.Unlock()
			return v
		}
		s.mu.Unlock()

		raw, err := s.eximClient.FetchUSDRate(candidate)
		if err != nil || raw == 0 {
			continue
		}
		rate, parseErr := numeric.FromString(fmt.Sprintf("%g", raw))
		if parseErr != nil {
			continue
		}
		s.mu.Lock()
		s.cachedRates[candidate] = rate
		s.mu.Unlock()
		return rate
	}
	return numeric.Zero
}
