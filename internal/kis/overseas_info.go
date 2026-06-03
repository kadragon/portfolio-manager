package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	// PrdtClsfName is the product classification name (상품분류명); contains "ETF"
	// for exchange-traded funds.
	PrdtClsfName string
	// EtfRiskCd is the overseas-ETF risk indicator code (해외주식ETF위험지표코드);
	// non-empty only for ETFs.
	EtfRiskCd string
	// EtpTrackMul is the ETP tracking multiple (ETP추적수익율배수); non-zero only
	// for ETPs (ETF/ETN).
	EtpTrackMul string
}

// AssetClass returns "etf" or "stock" derived from the info's classification fields.
func (i OverseasStockInfo) AssetClass() string {
	return ClassifyOverseasAssetClass(i.PrdtClsfName, i.EtfRiskCd, i.EtpTrackMul)
}

// ClassifyOverseasAssetClass maps KIS overseas search-info fields to "etf" or
// "stock". An ETF reports "ETF" in its product classification name, carries an
// overseas-ETF risk indicator code, or has a non-zero ETP tracking multiple.
func ClassifyOverseasAssetClass(prdtClsfName, etfRiskCd, etpTrackMul string) string {
	if strings.Contains(strings.ToUpper(prdtClsfName), "ETF") {
		return "etf"
	}
	if strings.TrimSpace(etfRiskCd) != "" {
		return "etf"
	}
	if m := strings.TrimSpace(etpTrackMul); m != "" && m != "0" {
		return "etf"
	}
	return "stock"
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
		Pdno:         pdno,
		PrdtTypeCd:   prdtTypeCd,
		Excd:         excd,
		Name:         name,
		PrdtClsfName: output["prdt_clsf_name"],
		EtfRiskCd:    output["ovrs_stck_etf_risk_drtp_cd"],
		EtpTrackMul:  output["etp_chas_erng_rt_dbnb"],
	}, nil
}

// ClassifyAssetClass looks up an overseas ticker and returns "etf" or "stock".
// excd is the order-form exchange code (e.g. "NAS"/"NYS"/"AMS").
func (c *OverseasInfoClient) ClassifyAssetClass(excd, ticker string) (string, error) {
	info, err := c.FetchBasicInfo(excd, ticker)
	if err != nil {
		return "", err
	}
	return info.AssetClass(), nil
}
