package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var excdToPrdtTypeCd = map[string]string{
	"NAS": "512",
	"NYS": "513",
	"AMS": "529",
}

// OverseasStockInfo holds basic info for an overseas stock.
type OverseasStockInfo struct {
	Pdno       string
	PrdtTypeCd string
	Excd       string
	Name       string
}

// OverseasInfoClient fetches stock name and product info via overseas search-info.
type OverseasInfoClient struct {
	HTTP      *http.Client
	BaseURL   string
	AppKey    string
	AppSecret string
	TrID      string
	CustType  string
	Manager   *TokenManager
}

// FetchBasicInfo calls CTPF1702R (or env-configured tr_id) to look up overseas stock metadata.
// excd is the order-form exchange code (e.g. "NAS", "NYS", "AMS").
func (c *OverseasInfoClient) FetchBasicInfo(excd, ticker string) (OverseasStockInfo, error) {
	prdtTypeCd := excdToPrdtTypeCd[excd]
	if prdtTypeCd == "" {
		prdtTypeCd = "512"
	}
	token, err := c.Manager.GetToken()
	if err != nil {
		return OverseasStockInfo{}, err
	}
	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/overseas-price/v1/quotations/search-info",
		map[string]string{
			"PRDT_TYPE_CD": prdtTypeCd,
			"PDNO":         ticker,
		},
		BuildHeaders(token, c.AppKey, c.AppSecret, c.TrID, c.CustType),
		c.Manager, c.AppKey, c.AppSecret, c.TrID, c.CustType,
	)
	if err != nil {
		return OverseasStockInfo{}, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return OverseasStockInfo{}, err
	}

	// Check for KIS business error.
	var meta struct {
		RtCd string `json:"rt_cd"`
		Msg1 string `json:"msg1"`
		Msg  string `json:"msg_cd"`
	}
	_ = json.Unmarshal(body, &meta)
	if meta.RtCd != "" && meta.RtCd != "0" {
		return OverseasStockInfo{}, fmt.Errorf("KIS overseas info [%s]: %s", meta.Msg, meta.Msg1)
	}

	output, err := extractOutputObject(raw)
	if err != nil {
		return OverseasStockInfo{}, fmt.Errorf("KIS overseas info %s: %w", ticker, err)
	}

	name := output["prdt_name"]
	if name == "" {
		name = output["prdt_eng_name"]
	}
	if name == "" {
		name = output["prdt_name120"]
	}
	if name == "" {
		name = output["prdt_abrv_name"]
	}

	pdno := output["pdno"]
	if pdno == "" {
		pdno = ticker
	}

	return OverseasStockInfo{
		Pdno:       pdno,
		PrdtTypeCd: prdtTypeCd,
		Excd:       excd,
		Name:       name,
	}, nil
}
