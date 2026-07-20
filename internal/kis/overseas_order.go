package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OverseasOrderClient places buy/sell orders on overseas exchanges via KIS.
type OverseasOrderClient struct {
	HTTP       *http.Client
	BaseURL    string
	AppKey     string
	AppSecret  string
	CANO       string // account number (8 digits)
	AcntPrdtCd string // account product code (2 digits)
	CustType   string
	Env        string
	Manager    *TokenManager
}

// PlaceOrder places an overseas order and returns the raw KIS response.
// exchange must be the KIS order-form code: NASD, NYSE, AMEX. price is the
// limit price: empty places a market order (ORD_DVSN 01, OVRS_ORD_UNPR 0); a
// non-empty value places a limit order (ORD_DVSN 00, OVRS_ORD_UNPR price).
func (c *OverseasOrderClient) PlaceOrder(ctx context.Context, ticker, side string, quantity int, exchange string, price string) (map[string]any, error) {
	trID, err := TrIDForEnv(c.Env, overseasOrderTrID(side, false), overseasOrderTrID(side, true))
	if err != nil {
		return nil, err
	}

	token, err := c.Manager.GetToken()
	if err != nil {
		return nil, err
	}

	ordDvsn, ordUnpr := "01", "0"
	if strings.TrimSpace(price) != "" {
		ordDvsn, ordUnpr = "00", strings.TrimSpace(price)
	}

	payload := map[string]string{
		"CANO":          c.CANO,
		"ACNT_PRDT_CD":  c.AcntPrdtCd,
		"OVRS_EXCG_CD":  exchange,
		"PDNO":          ticker,
		"ORD_QTY":       fmt.Sprintf("%d", quantity),
		"OVRS_ORD_UNPR": ordUnpr,
		"ORD_DVSN":      ordDvsn,
	}

	headers := BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType)
	headers["content-type"] = "application/json; charset=utf-8"

	body, err := postWithRetry(ctx, c.HTTP, c.BaseURL+"/uapi/overseas-stock/v1/trading/order", payload, headers, c.Manager, c.AppKey, c.AppSecret, trID, c.CustType)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func overseasOrderTrID(side string, demo bool) string {
	if side == "buy" {
		if demo {
			return "VTTT1002U"
		}
		return "TTTT1002U"
	}
	if demo {
		return "VTTT1006U"
	}
	return "TTTT1006U"
}
