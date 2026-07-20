package kis

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shopspring/decimal"
)

// DomesticBuyableClient queries KIS 주식 매수가능조회 (available buying power)
// via the inquire-psbl-order endpoint.
type DomesticBuyableClient struct {
	HTTP      *http.Client
	BaseURL   string
	AppKey    string
	AppSecret string
	CustType  string
	Env       string
	Manager   *TokenManager
}

// BuyableAmount is the parsed 매수가능조회 response. Amounts are in KRW.
type BuyableAmount struct {
	OrderableCash       decimal.Decimal // ord_psbl_cash 주문가능현금
	OrderableSubstitute decimal.Decimal // ord_psbl_sbst 주문가능대용
	ReusableAmount      decimal.Decimal // ruse_psbl_amt 재사용가능금액
	NrcvbBuyAmount      decimal.Decimal // nrcvb_buy_amt 미수없는매수금액
	NrcvbBuyQty         decimal.Decimal // nrcvb_buy_qty 미수없는매수수량
	MaxBuyAmount        decimal.Decimal // max_buy_amt 최대매수금액
	MaxBuyQty           decimal.Decimal // max_buy_qty 최대매수수량
}

// FetchBuyable queries available buying power for the account. pdno/ordUnpr are
// optional: an empty pdno with ordDvsn "01" (시장가) returns the cash figures
// only; passing a ticker + unit price fills in the max-buy quantity fields.
// Defaults: ordDvsn "01" (시장가), ordUnpr "0".
func (c *DomesticBuyableClient) FetchBuyable(cano, acntPrdtCd, pdno, ordUnpr, ordDvsn string) (BuyableAmount, error) {
	trID, err := TrIDForEnv(c.Env, "TTTC8908R", "VTTC8908R")
	if err != nil {
		return BuyableAmount{}, err
	}
	if ordDvsn == "" {
		ordDvsn = "01"
	}
	if ordUnpr == "" {
		ordUnpr = "0"
	}

	token, err := c.Manager.GetToken()
	if err != nil {
		return BuyableAmount{}, err
	}
	params := map[string]string{
		"CANO":                 cano,
		"ACNT_PRDT_CD":         acntPrdtCd,
		"PDNO":                 pdno,
		"ORD_UNPR":             ordUnpr,
		"ORD_DVSN":             ordDvsn,
		"CMA_EVLU_AMT_ICLD_YN": "N",
		"OVRS_ICLD_YN":         "N",
	}
	headers := BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType)

	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/domestic-stock/v1/trading/inquire-psbl-order",
		params, headers,
		c.Manager, c.AppKey, c.AppSecret, trID, c.CustType,
	)
	if err != nil {
		return BuyableAmount{}, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return BuyableAmount{}, fmt.Errorf("kis buyable: json unmarshal: %w", err)
	}
	if err := raiseBizError(data); err != nil {
		return BuyableAmount{}, err
	}

	out := firstMap(toSliceOfMaps(data["output"]))
	return BuyableAmount{
		OrderableCash:       parseDecimal(strVal(out, "ord_psbl_cash")),
		OrderableSubstitute: parseDecimal(strVal(out, "ord_psbl_sbst")),
		ReusableAmount:      parseDecimal(strVal(out, "ruse_psbl_amt")),
		NrcvbBuyAmount:      parseDecimal(strVal(out, "nrcvb_buy_amt")),
		NrcvbBuyQty:         parseDecimal(strVal(out, "nrcvb_buy_qty")),
		MaxBuyAmount:        parseDecimal(strVal(out, "max_buy_amt")),
		MaxBuyQty:           parseDecimal(strVal(out, "max_buy_qty")),
	}, nil
}

// firstMap returns the first element of maps, or nil (strVal reads nil maps safely).
func firstMap(maps []map[string]any) map[string]any {
	if len(maps) > 0 {
		return maps[0]
	}
	return nil
}
