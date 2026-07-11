package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/datex"
)

const overseasDateBufferDays = 7

// OverseasPriceClient fetches current and historical prices for overseas stocks.
type OverseasPriceClient struct {
	HTTP      *http.Client
	BaseURL   string
	AppKey    string
	AppSecret string
	CustType  string
	Env       string
	Manager   *TokenManager
}

// FetchCurrentPrice calls HHDFS00000300 (overseas-price inquire) for ticker on excd exchange.
// excd is the canonical code (e.g. "NASD", "NYSE", "AMEX"); converted to short form via shortExchangeCode.
func (c *OverseasPriceClient) FetchCurrentPrice(excd, ticker string) (KisPriceQuote, error) {
	trID, err := TrIDForEnv(c.Env, "HHDFS00000300", "HHDFS00000300")
	if err != nil {
		return KisPriceQuote{}, err
	}
	token, err := c.Manager.GetToken()
	if err != nil {
		return KisPriceQuote{}, err
	}
	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/overseas-price/v1/quotations/price",
		map[string]string{
			"AUTH": "",
			"EXCD": shortExchangeCode(excd),
			"SYMB": ticker,
		},
		BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType),
		c.Manager, c.AppKey, c.AppSecret, trID, c.CustType,
	)
	if err != nil {
		return KisPriceQuote{}, err
	}
	if rtCd, msgCd, msg1 := ParseKISStatus(body); rtCd != "" && rtCd != "0" {
		return KisPriceQuote{}, fmt.Errorf("KIS price [%s@%s] rt_cd=%s %s: %s", ticker, excd, rtCd, msgCd, msg1)
	}
	return ParseUSPrice(body, ticker, excd), nil
}

// FetchHistoricalClose calls HHDFS76240000 (overseas dailyprice) for a past date.
func (c *OverseasPriceClient) FetchHistoricalClose(excd, ticker string, targetDate datex.Date) (float64, error) {
	trID, err := TrIDForEnv(c.Env, "HHDFS76240000", "HHDFS76240000")
	if err != nil {
		return 0, err
	}
	token, err := c.Manager.GetToken()
	if err != nil {
		return 0, err
	}
	// Set BYMD after targetDate so the target falls within the returned range.
	bymd := targetDate.Time.AddDate(0, 0, overseasDateBufferDays)
	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/overseas-price/v1/quotations/dailyprice",
		map[string]string{
			"AUTH": "",
			"EXCD": shortExchangeCode(excd),
			"SYMB": ticker,
			"GUBN": "0",
			"BYMD": bymd.Format("20060102"),
			"MODP": "0",
		},
		BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType),
		nil, c.AppKey, c.AppSecret, trID, c.CustType,
	)
	if err != nil {
		return 0, err
	}
	if rtCd, msgCd, msg1 := ParseKISStatus(body); rtCd != "" && rtCd != "0" {
		return 0, fmt.Errorf("KIS dailyprice [%s@%s] rt_cd=%s %s: %s", ticker, excd, rtCd, msgCd, msg1)
	}
	return parseOverseasHistorical(body, targetDate.Time.Format("20060102")), nil
}

// FetchHistoricalRange calls HHDFS76240000 (overseas dailyprice) and returns every daily
// close KIS reports at or after startDate, up to endDate. This endpoint only takes a single
// end-anchor date (BYMD) and walks backward, so results after endDate are filtered client-side
// and results are capped at whatever row count KIS returns per call (~100).
func (c *OverseasPriceClient) FetchHistoricalRange(excd, ticker string, startDate, endDate datex.Date) ([]HistoricalPoint, error) {
	trID, err := TrIDForEnv(c.Env, "HHDFS76240000", "HHDFS76240000")
	if err != nil {
		return nil, err
	}
	token, err := c.Manager.GetToken()
	if err != nil {
		return nil, err
	}
	// Set BYMD after endDate so endDate itself falls within the returned range, same as FetchHistoricalClose.
	bymd := endDate.Time.AddDate(0, 0, overseasDateBufferDays)
	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/overseas-price/v1/quotations/dailyprice",
		map[string]string{
			"AUTH": "",
			"EXCD": shortExchangeCode(excd),
			"SYMB": ticker,
			"GUBN": "0",
			"BYMD": bymd.Format("20060102"),
			"MODP": "0",
		},
		BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType),
		nil, c.AppKey, c.AppSecret, trID, c.CustType,
	)
	if err != nil {
		return nil, err
	}
	if rtCd, msgCd, msg1 := ParseKISStatus(body); rtCd != "" && rtCd != "0" {
		return nil, fmt.Errorf("KIS dailyprice [%s@%s] rt_cd=%s %s: %s", ticker, excd, rtCd, msgCd, msg1)
	}
	points := parseOverseasHistoricalRange(body)
	filtered := make([]HistoricalPoint, 0, len(points))
	for _, p := range points {
		if p.Date.ISO() < startDate.ISO() || p.Date.ISO() > endDate.ISO() {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered, nil
}

func parseOverseasHistoricalRange(data []byte) []HistoricalPoint {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}

	var items []map[string]string
	if blob, ok := raw["output2"]; ok {
		_ = json.Unmarshal(blob, &items)
	}

	points := make([]HistoricalPoint, 0, len(items))
	for _, item := range items {
		d, err := parseKISDate(item["xymd"])
		if err != nil {
			continue
		}
		price := parseFloat(strings.TrimSpace(item["clos"]))
		if price <= 0 {
			continue
		}
		points = append(points, HistoricalPoint{Date: d, Price: price})
	}
	return points
}

func parseOverseasHistorical(data []byte, targetStr string) float64 {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return 0
	}

	var items []map[string]string
	if blob, ok := raw["output2"]; ok {
		if json.Unmarshal(blob, &items) != nil || len(items) == 0 {
			// Single object fallback
			var single map[string]string
			if json.Unmarshal(blob, &single) == nil {
				items = []map[string]string{single}
			}
		}
	}

	for _, item := range items {
		if item["xymd"] == targetStr {
			return parseFloat(strings.TrimSpace(item["clos"]))
		}
	}
	// Fallback to most-recent available.
	if len(items) > 0 {
		return parseFloat(strings.TrimSpace(items[0]["clos"]))
	}
	return 0
}
