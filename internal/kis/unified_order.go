package kis

import (
	"context"
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
// price is the limit price (empty = market order); see the underlying clients.
func (c *UnifiedOrderClient) PlaceOrder(ctx context.Context, ticker, side string, quantity int, exchange string, price string) (map[string]any, error) {
	if IsDomesticTicker(ticker) {
		if c.Domestic == nil {
			return nil, fmt.Errorf("domestic order client not configured")
		}
		return c.Domestic.PlaceOrder(ctx, ticker, side, quantity, "", price)
	}
	if c.Overseas == nil {
		return nil, fmt.Errorf("overseas order client not configured")
	}
	ex := normalizeOrderExchange(exchange)
	return c.Overseas.PlaceOrder(ctx, ticker, side, quantity, ex, price)
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
