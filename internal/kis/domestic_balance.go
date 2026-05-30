package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/shopspring/decimal"
)

// DomesticBalanceClient fetches domestic account balance from the KIS API.
type DomesticBalanceClient struct {
	HTTP      *http.Client
	BaseURL   string
	AppKey    string
	AppSecret string
	CustType  string
	Env       string
	Manager   *TokenManager
}

// FetchAccountSnapshot fetches cash balance and holdings via paginated KIS balance API.
// Holdings in the returned snapshot are sorted by ticker (matches Python's sorted()).
func (c *DomesticBalanceClient) FetchAccountSnapshot(cano, acntPrdtCd string) (models.KisAccountSnapshot, error) {
	trID, err := balanceTrIDForEnv(c.Env)
	if err != nil {
		return models.KisAccountSnapshot{}, err
	}

	fk100 := ""
	nk100 := ""
	trCont := ""
	cashBalance := numeric.Zero
	holdingQty := map[string]decimal.Decimal{} // ticker → aggregated qty
	holdingName := map[string]string{}         // ticker → first seen name

	for {
		params := map[string]string{
			"CANO":                  cano,
			"ACNT_PRDT_CD":          acntPrdtCd,
			"AFHR_FLPR_YN":          "N",
			"OFL_YN":                "",
			"INQR_DVSN":             "01",
			"UNPR_DVSN":             "01",
			"FUND_STTL_ICLD_YN":     "N",
			"FNCG_AMT_AUTO_RDPT_YN": "N",
			"PRCS_DVSN":             "00",
			"CTX_AREA_FK100":        fk100,
			"CTX_AREA_NK100":        nk100,
		}
		token, err := c.Manager.GetToken()
		if err != nil {
			return models.KisAccountSnapshot{}, err
		}
		headers := BuildHeaders(token, c.AppKey, c.AppSecret, trID, c.CustType)
		if trCont != "" {
			headers["tr_cont"] = trCont
		}

		body, respHeaders, err := GetWithRetryFull(
			c.HTTP,
			c.BaseURL+"/uapi/domestic-stock/v1/trading/inquire-balance",
			params, headers,
			c.Manager, c.AppKey, c.AppSecret, trID, c.CustType,
		)
		if err != nil {
			return models.KisAccountSnapshot{}, err
		}

		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			return models.KisAccountSnapshot{}, fmt.Errorf("kis balance: json unmarshal: %w", err)
		}
		if err := raiseBizError(data); err != nil {
			return models.KisAccountSnapshot{}, err
		}

		output1 := toSliceOfMaps(data["output1"])
		output2 := toSliceOfMaps(data["output2"])

		for _, item := range output1 {
			ticker := strings.TrimSpace(strVal(item, "pdno"))
			qty := parseDecimal(strVal(item, "hldg_qty"))
			if ticker == "" || !qty.IsPositive() {
				continue
			}
			if prev, ok := holdingQty[ticker]; ok {
				holdingQty[ticker] = prev.Add(qty)
			} else {
				holdingQty[ticker] = qty
			}
			if _, hasName := holdingName[ticker]; !hasName {
				name := strings.TrimSpace(strVal(item, "prdt_name"))
				if name != "" {
					holdingName[ticker] = name
				}
			}
		}

		if len(output2) > 0 {
			cashBalance = parseCashBalance(output2[0])
		}

		headerTrCont := respHeaders.Get("tr_cont")
		if headerTrCont != "M" && headerTrCont != "F" {
			break
		}
		fk100 = strings.TrimSpace(strVal(data, "ctx_area_fk100"))
		nk100 = strings.TrimSpace(strVal(data, "ctx_area_nk100"))
		trCont = "N"
	}

	tickers := make([]string, 0, len(holdingQty))
	for t := range holdingQty {
		tickers = append(tickers, t)
	}
	sort.Strings(tickers)

	holdings := make([]models.KisHoldingPosition, 0, len(tickers))
	for _, t := range tickers {
		holdings = append(holdings, models.KisHoldingPosition{
			Ticker:   t,
			Quantity: numeric.Wrap(holdingQty[t]),
			Name:     holdingName[t],
		})
	}

	return models.KisAccountSnapshot{CashBalance: cashBalance, Holdings: holdings}, nil
}

func balanceTrIDForEnv(env string) (string, error) {
	return TrIDForEnv(env, "TTTC8434R", "VTTC8434R")
}

func raiseBizError(data map[string]any) error {
	rtCd := strings.TrimSpace(strVal(data, "rt_cd"))
	if rtCd == "" || rtCd == "0" {
		return nil
	}
	code := strings.TrimSpace(strVal(data, "msg_cd"))
	msg := strings.TrimSpace(strVal(data, "msg1"))
	return fmt.Errorf("KIS API error %s: %s", code, msg)
}

// parseCashBalance checks keys in order: dnca_tot_amt → ord_psbl_cash → tot_dnca_amt.
// A present key with value "0" returns 0 (key-existence, not truthiness).
func parseCashBalance(row map[string]any) numeric.Decimal {
	for _, key := range []string{"dnca_tot_amt", "ord_psbl_cash", "tot_dnca_amt"} {
		if v, ok := row[key]; ok {
			return numeric.Wrap(parseDecimal(fmt.Sprintf("%v", v)))
		}
	}
	return numeric.Zero
}

func parseDecimal(s string) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func toSliceOfMaps(v any) []map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return []map[string]any{m}
	}
	if s, ok := v.([]any); ok {
		result := make([]map[string]any, 0, len(s))
		for _, item := range s {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result
	}
	return nil
}
