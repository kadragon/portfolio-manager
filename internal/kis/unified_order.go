package kis

import (
	"fmt"
)

// orderExchangeMap converts price-form overseas exchange codes to order-form.
var orderExchangeMap = map[string]string{
	"NAS": "NASD",
	"NYS": "NYSE",
	"AMS": "AMEX",
}

const defaultOrderExchange = "NASD"

// UnifiedOrderClient routes orders to domestic or overseas KIS clients.
type UnifiedOrderClient struct {
	Domestic *DomesticOrderClient
	Overseas *OverseasOrderClient
}

// PlaceOrder routes to domestic or overseas client based on ticker length.
// exchange should be the order-form code (NASD/NYSE/AMEX) or price-form (NAS/NYS/AMS).
func (c *UnifiedOrderClient) PlaceOrder(ticker, side string, quantity int, exchange string) (map[string]any, error) {
	if IsDomesticTicker(ticker) {
		if c.Domestic == nil {
			return nil, fmt.Errorf("domestic order client not configured")
		}
		return c.Domestic.PlaceOrder(ticker, side, quantity, "")
	}
	if c.Overseas == nil {
		return nil, fmt.Errorf("overseas order client not configured")
	}
	ex := normalizeOrderExchange(exchange)
	return c.Overseas.PlaceOrder(ticker, side, quantity, ex)
}

func normalizeOrderExchange(exchange string) string {
	if exchange == "" {
		return defaultOrderExchange
	}
	if mapped, ok := orderExchangeMap[exchange]; ok {
		return mapped
	}
	return exchange
}
