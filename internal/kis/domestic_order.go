package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
// exchange is ignored for domestic orders (always KRX). price is the limit
// price: empty places a market order (ORD_DVSN 01, ORD_UNPR 0); a non-empty
// value places a limit order (ORD_DVSN 00, ORD_UNPR price). A limit order
// reserves exactly price×qty of buying power, whereas a market buy reserves at
// the daily upper limit — so a buy sized to fit available cash must be a limit
// order to clear KIS's "주문가능금액 초과" check.
func (c *DomesticOrderClient) PlaceOrder(ctx context.Context, ticker, side string, quantity int, _ string, price string) (map[string]any, error) {
	trID, err := TrIDForEnv(c.Env, domesticOrderTrID(side, false), domesticOrderTrID(side, true))
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
		"CANO":            c.CANO,
		"ACNT_PRDT_CD":    c.AcntPrdtCd,
		"PDNO":            ticker,
		"ORD_DVSN":        ordDvsn,
		"ORD_QTY":         fmt.Sprintf("%d", quantity),
		"ORD_UNPR":        ordUnpr,
		"EXCG_ID_DVSN_CD": "KRX",
	}

	headers := BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType)
	headers["content-type"] = "application/json; charset=utf-8"

	body, err := postWithRetry(ctx, c.HTTP, c.BaseURL+"/uapi/domestic-stock/v1/trading/order-cash", payload, headers, c.Manager, c.AppKey, c.AppSecret, trID, c.CustType)
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
