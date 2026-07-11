package kis

import (
	"fmt"
	"log"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// Compile-time assertion that UnifiedPriceClient satisfies services.PriceClient.
var _ services.PriceClient = (*UnifiedPriceClient)(nil)

// prioritizedExchanges returns canonical exchange codes to try in order, preferring the given one.
// These are the internal canonical codes (NASD/NYSE/AMEX), which match the order-endpoint form.
// The price endpoint uses the short form (NAS/NYS/AMS); OverseasPriceClient.FetchCurrentPrice
// converts them at the wire boundary via priceEndpointEXCD.
func prioritizedExchanges(preferred string) []string {
	all := []string{"NASD", "NYSE", "AMEX"}
	if preferred == "" {
		return all
	}
	// Normalize legacy short-form codes (NAS/NYS/AMS) — pre-migration data stored the
	// price-endpoint's 3-letter form directly in stocks.exchange, which never matches
	// the canonical set below and silently falls back to NASD-first for every ticker.
	if mapped, ok := orderExchangeMap[preferred]; ok {
		preferred = mapped
	}
	for _, e := range all {
		if e == preferred {
			result := []string{preferred}
			for _, other := range all {
				if other != preferred {
					result = append(result, other)
				}
			}
			return result
		}
	}
	return all
}

// UnifiedPriceClient implements services.PriceClient by routing to the
// domestic or overseas KIS clients based on ticker length.
type UnifiedPriceClient struct {
	Domestic     *DomesticPriceClient
	Overseas     *OverseasPriceClient
	DomesticInfo *DomesticInfoClient // optional — enriches name on cache miss
	OverseasInfo *OverseasInfoClient // optional — enriches name on cache miss
	PrdtTypeCd   string              // domestic info product type, default "300"
}

// GetPrice returns the current quote for ticker. preferredExchange is the
// canonical/order-form code ("NASD", "NYSE", "AMEX") or empty for domestic/auto.
// The underlying price endpoint uses short codes (NAS/NYS/AMS); conversion happens inside
// OverseasPriceClient.
func (c *UnifiedPriceClient) GetPrice(ticker string, preferredExchange string) (services.PriceQuote, error) {
	if IsDomesticTicker(ticker) {
		return c.getDomesticPrice(ticker)
	}
	return c.getOverseasPrice(ticker, preferredExchange)
}

// GetHistoricalClose returns the closing price for ticker on date.
func (c *UnifiedPriceClient) GetHistoricalClose(ticker string, date datex.Date, preferredExchange string) (float64, error) {
	if IsDomesticTicker(ticker) {
		price, err := c.Domestic.FetchHistoricalClose(ticker, date)
		if err != nil {
			log.Printf("KIS: domestic historical close %s: %v", ticker, err)
			return 0, nil
		}
		return price, nil
	}
	return c.getOverseasHistorical(ticker, date, preferredExchange)
}

func (c *UnifiedPriceClient) getDomesticPrice(ticker string) (services.PriceQuote, error) {
	quote, err := c.Domestic.FetchCurrentPrice("J", ticker)
	if err != nil {
		log.Printf("KIS: domestic price %s: %v", ticker, err)
		return services.PriceQuote{Symbol: ticker, Currency: "KRW"}, nil
	}
	// Enrich name from info client when missing.
	if quote.Name == "" && c.DomesticInfo != nil {
		prdtTypeCd := c.PrdtTypeCd
		if prdtTypeCd == "" {
			prdtTypeCd = "300"
		}
		info, infoErr := c.DomesticInfo.FetchBasicInfo(prdtTypeCd, ticker)
		if infoErr == nil {
			quote.Name = info.Name
		}
	}
	return services.PriceQuote{
		Symbol:   quote.Symbol,
		Name:     quote.Name,
		Price:    quote.Price,
		Currency: quote.Currency,
		Exchange: quote.Exchange,
	}, nil
}

func (c *UnifiedPriceClient) getOverseasPrice(ticker, preferredExchange string) (services.PriceQuote, error) {
	exchanges := prioritizedExchanges(preferredExchange)
	var best *KisPriceQuote
	for _, excd := range exchanges {
		quote, err := c.Overseas.FetchCurrentPrice(excd, ticker)
		if err != nil {
			log.Printf("KIS: overseas price %s@%s: %v", ticker, excd, err)
			continue
		}
		if best == nil {
			best = &quote
		}
		if quote.Name != "" {
			best = &quote
			break
		}
		if best.Price == 0 && quote.Price > 0 {
			best = &quote
		}
		// Stop early only once the preferred exchange actually yields a usable price —
		// a success-but-empty response (ticker not listed there) must fall through to
		// the other exchanges rather than being accepted as the final answer.
		if preferredExchange != "" && quote.Price > 0 {
			break
		}
	}
	if best == nil {
		return services.PriceQuote{Symbol: ticker, Currency: "USD"}, nil
	}
	// Enrich name from info client when missing.
	if best.Name == "" && c.OverseasInfo != nil {
		excdForInfo := best.Exchange
		if excdForInfo == "" && len(exchanges) > 0 {
			excdForInfo = exchanges[0]
		}
		// search-info endpoint is quotation-family; requires short 3-letter code.
		orderExcd := shortExchangeCode(excdForInfo)
		info, infoErr := c.OverseasInfo.FetchBasicInfo(orderExcd, ticker)
		if infoErr == nil {
			best.Name = info.Name
		}
	}
	return services.PriceQuote{
		Symbol:   best.Symbol,
		Name:     best.Name,
		Price:    best.Price,
		Currency: best.Currency,
		Exchange: best.Exchange,
	}, nil
}

// GetHistoricalRange returns every daily close available for ticker within [start, end].
func (c *UnifiedPriceClient) GetHistoricalRange(ticker string, start, end datex.Date, preferredExchange string) ([]services.HistoricalPricePoint, error) {
	if IsDomesticTicker(ticker) {
		points, err := c.Domestic.FetchHistoricalRange(ticker, start, end)
		if err != nil {
			return nil, fmt.Errorf("KIS domestic historical range %s: %w", ticker, err)
		}
		return toHistoricalPricePoints(points), nil
	}
	return c.getOverseasHistoricalRange(ticker, start, end, preferredExchange)
}

func (c *UnifiedPriceClient) getOverseasHistoricalRange(ticker string, start, end datex.Date, preferredExchange string) ([]services.HistoricalPricePoint, error) {
	exchanges := prioritizedExchanges(preferredExchange)
	var lastErr error
	for _, excd := range exchanges {
		points, err := c.Overseas.FetchHistoricalRange(excd, ticker, start, end)
		if err != nil {
			lastErr = err
			continue
		}
		if len(points) > 0 {
			return toHistoricalPricePoints(points), nil
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("KIS overseas historical range %s: %w", ticker, lastErr)
	}
	return nil, nil
}

func toHistoricalPricePoints(points []HistoricalPoint) []services.HistoricalPricePoint {
	out := make([]services.HistoricalPricePoint, 0, len(points))
	for _, p := range points {
		out = append(out, services.HistoricalPricePoint{Date: p.Date, Price: p.Price})
	}
	return out
}

func (c *UnifiedPriceClient) getOverseasHistorical(ticker string, date datex.Date, preferredExchange string) (float64, error) {
	if preferredExchange != "" {
		price, err := c.Overseas.FetchHistoricalClose(preferredExchange, ticker, date)
		if err != nil {
			log.Printf("KIS: overseas historical close %s@%s: %v", ticker, preferredExchange, err)
		} else if price > 0 {
			return price, nil
		}
	}
	exchanges := prioritizedExchanges(preferredExchange)
	for _, excd := range exchanges {
		if excd == preferredExchange {
			continue
		}
		price, err := c.Overseas.FetchHistoricalClose(excd, ticker, date)
		if err != nil {
			log.Printf("KIS: overseas historical close %s@%s: %v", ticker, excd, err)
			continue
		}
		if price > 0 {
			return price, nil
		}
	}
	return 0, nil
}

// shortExchangeCode converts canonical 4-letter codes (NASD/NYSE/AMEX) to the 3-letter
// form (NAS/NYS/AMS) required by KIS quotation endpoints (price, dailyprice, search-info).
// The order endpoint uses the 4-letter canonical form; these two families use opposite conventions.
func shortExchangeCode(excd string) string {
	switch excd {
	case "NASD":
		return "NAS"
	case "NYSE":
		return "NYS"
	case "AMEX":
		return "AMS"
	default:
		return excd
	}
}
