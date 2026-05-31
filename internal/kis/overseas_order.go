package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// OverseasOrderClient places buy/sell orders on overseas exchanges via KIS.
type OverseasOrderClient struct {
	HTTP      *http.Client
	BaseURL   string
	AppKey    string
	AppSecret string
	CustType  string
	Env       string
	Manager   *TokenManager
}

// PlaceOrder places an overseas order and returns the raw KIS response.
// exchange must be the KIS order-form code: NASD, NYSE, AMEX.
func (c *OverseasOrderClient) PlaceOrder(ticker, side string, quantity int, exchange string) (map[string]any, error) {
	trID, err := TrIDForEnv(c.Env, overseasOrderTrID(side, false), overseasOrderTrID(side, true))
	if err != nil {
		return nil, err
	}

	token, err := c.Manager.GetToken()
	if err != nil {
		return nil, err
	}

	payload := map[string]string{
		"OVRS_EXCG_CD":  exchange,
		"PDNO":          ticker,
		"ORD_QTY":       fmt.Sprintf("%d", quantity),
		"OVRS_ORD_UNPR": "0",
		"ORD_DVSN":      "01",
	}

	headers := BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType)
	headers["content-type"] = "application/json; charset=utf-8"

	body, err := postWithRetry(c.HTTP, c.BaseURL+"/uapi/overseas-stock/v1/trading/order", payload, headers, c.Manager, c.AppKey, c.AppSecret, trID, c.CustType)
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
