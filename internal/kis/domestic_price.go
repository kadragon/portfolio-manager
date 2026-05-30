package kis

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/datex"
)

const domesticDateBufferDays = 7

// DomesticPriceClient fetches current and historical prices for KOSPI/KOSDAQ stocks.
type DomesticPriceClient struct {
	HTTP      *http.Client
	BaseURL   string
	AppKey    string
	AppSecret string
	CustType  string
	Env       string
	Manager   *TokenManager
}

// FetchCurrentPrice calls FHKST01010100 (inquire-price) for a domestic ticker.
// fidCondMrktDivCode is typically "J" for KOSPI/KOSDAQ.
func (c *DomesticPriceClient) FetchCurrentPrice(fidCondMrktDivCode, ticker string) (KisPriceQuote, error) {
	trID, err := TrIDForEnv(c.Env, "FHKST01010100", "FHKST01010100")
	if err != nil {
		return KisPriceQuote{}, err
	}
	token, err := c.Manager.GetToken()
	if err != nil {
		return KisPriceQuote{}, err
	}
	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/domestic-stock/v1/quotations/inquire-price",
		map[string]string{
			"FID_COND_MRKT_DIV_CODE": fidCondMrktDivCode,
			"FID_INPUT_ISCD":         ticker,
		},
		BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType),
		c.Manager, c.AppKey, c.AppSecret, trID, c.CustType,
	)
	if err != nil {
		return KisPriceQuote{}, err
	}
	return ParseKoreaPrice(body, ticker), nil
}

// FetchHistoricalClose calls FHKST03010100 (inquire-daily-itemchartprice) for a past date.
func (c *DomesticPriceClient) FetchHistoricalClose(ticker string, targetDate datex.Date) (float64, error) {
	trID, err := TrIDForEnv(c.Env, "FHKST03010100", "FHKST03010100")
	if err != nil {
		return 0, err
	}
	token, err := c.Manager.GetToken()
	if err != nil {
		return 0, err
	}
	startDate := targetDate.Time.AddDate(0, 0, -domesticDateBufferDays)
	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice",
		map[string]string{
			"FID_COND_MRKT_DIV_CODE": "J",
			"FID_INPUT_ISCD":         ticker,
			"FID_INPUT_DATE_1":       startDate.Format("20060102"),
			"FID_INPUT_DATE_2":       targetDate.Time.Format("20060102"),
			"FID_PERIOD_DIV_CODE":    "D",
			"FID_ORG_ADJ_PRC":        "1",
		},
		BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType),
		nil, c.AppKey, c.AppSecret, trID, c.CustType,
	)
	if err != nil {
		return 0, err
	}
	return parseDomesticHistorical(body, targetDate.Time.Format("20060102")), nil
}

func parseDomesticHistorical(data []byte, targetStr string) float64 {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return 0
	}

	// API returns output2 for chart data; fall back to output if absent.
	var items []map[string]string
	for _, key := range []string{"output2", "output"} {
		blob, ok := raw[key]
		if !ok {
			continue
		}
		if json.Unmarshal(blob, &items) == nil && len(items) > 0 {
			break
		}
		// Single-object fallback
		var single map[string]string
		if json.Unmarshal(blob, &single) == nil {
			return parseDomesticClose(single)
		}
	}

	// Find the exact target date.
	for _, item := range items {
		if item["stck_bsop_date"] == targetStr {
			return parseDomesticClose(item)
		}
	}
	// Fallback to most-recent available.
	if len(items) > 0 {
		return parseDomesticClose(items[0])
	}
	return 0
}

func parseDomesticClose(item map[string]string) float64 {
	for _, key := range []string{"stck_clpr", "stck_prpr"} {
		if v := strings.TrimSpace(item[key]); v != "" && v != "0" {
			return parseFloat(v)
		}
	}
	return 0
}
