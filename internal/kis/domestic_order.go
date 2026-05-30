package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DomesticOrderClient places buy/sell orders on KOSPI/KOSDAQ via KIS.
type DomesticOrderClient struct {
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

// PlaceOrder places a domestic order and returns the raw KIS response.
// exchange is ignored for domestic orders (always KRX).
func (c *DomesticOrderClient) PlaceOrder(ticker, side string, quantity int, exchange string) (map[string]any, error) {
	trID, err := TrIDForEnv(c.Env, domesticOrderTrID(side, false), domesticOrderTrID(side, true))
	if err != nil {
		return nil, err
	}

	token, err := c.Manager.GetToken()
	if err != nil {
		return nil, err
	}

	// We need the current price for ord_unpr (limit price in market order form = 0).
	payload := map[string]string{
		"CANO":            c.CANO,
		"ACNT_PRDT_CD":    c.AcntPrdtCd,
		"PDNO":            ticker,
		"ORD_DVSN":        "00", // 지정가 (limit, required by KIS even for market orders in some envs)
		"ORD_QTY":         fmt.Sprintf("%d", quantity),
		"ORD_UNPR":        "0",
		"EXCG_ID_DVSN_CD": "KRX",
	}

	headers := BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType)
	headers["content-type"] = "application/json; charset=utf-8"

	body, err := postWithRetry(c.HTTP, c.BaseURL+"/uapi/domestic-stock/v1/trading/order-cash", payload, headers, c.Manager, c.AppKey, c.AppSecret, trID, c.CustType)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func domesticOrderTrID(side string, demo bool) string {
	if side == "buy" {
		if demo {
			return "VTTC0012U"
		}
		return "TTTC0012U"
	}
	if demo {
		return "VTTC0011U"
	}
	return "TTTC0011U"
}
