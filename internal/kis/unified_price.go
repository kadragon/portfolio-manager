package kis

import (
	"log"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// Compile-time assertion that UnifiedPriceClient satisfies services.PriceClient.
var _ services.PriceClient = (*UnifiedPriceClient)(nil)

// prioritizedExchanges returns exchanges to try in order, preferring the given one.
func prioritizedExchanges(preferred string) []string {
	all := []string{"NASD", "NYSE", "AMEX"}
	if preferred == "" {
		return all
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
// price-form code ("NASD", "NYSE", "AMEX") or empty for domestic/auto.
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
		if preferredExchange != "" {
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
		// OverseasInfoClient expects order-form exchange code (NAS, NYS, AMS).
		orderExcd := priceToOrderExchange(excdForInfo)
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

// priceToOrderExchange converts price-form to order-form exchange code.
var priceToOrderExchange = func() func(string) string {
	m := map[string]string{
		"NASD": "NAS",
		"NYSE": "NYS",
		"AMEX": "AMS",
	}
	return func(e string) string {
		if v, ok := m[e]; ok {
			return v
		}
		return e
	}
}()
