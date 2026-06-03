package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// DomesticStockInfo holds basic info for a KOSPI/KOSDAQ stock.
type DomesticStockInfo struct {
	Pdno       string
	PrdtTypeCd string
	MarketID   string
	Name       string
	// SctyGrpIDCd is the security group ID code (증권그룹ID코드): "EF"=ETF,
	// "ST"=주권, "EN"=ETN, etc. Primary signal for ETF classification.
	SctyGrpIDCd string
	// EtfDvsnCd is the ETF division code (ETF구분코드); set for ETFs.
	EtfDvsnCd string
}

// AssetClass returns "etf" or "stock" derived from the info's classification codes.
func (i DomesticStockInfo) AssetClass() string {
	return ClassifyDomesticAssetClass(i.SctyGrpIDCd, i.EtfDvsnCd)
}

// ClassifyDomesticAssetClass maps KIS search-stock-info classification codes to
// "etf" or "stock". A KOSPI/KOSDAQ-listed ETF reports scty_grp_id_cd "EF" (or the
// overseas-ETF variant "FE"); ETFs also carry a non-empty etf_dvsn_cd. Anything
// else (주권 "ST", ETN, etc.) is treated as a regular stock for eligibility.
func ClassifyDomesticAssetClass(sctyGrpIDCd, etfDvsnCd string) string {
	grp := strings.ToUpper(strings.TrimSpace(sctyGrpIDCd))
	if grp == "EF" || grp == "FE" {
		return "etf"
	}
	if d := strings.TrimSpace(etfDvsnCd); d != "" && d != "0" {
		return "etf"
	}
	return "stock"
}

// DomesticInfoClient fetches stock name and product info via search-stock-info.
type DomesticInfoClient struct {
	HTTP      *http.Client
	BaseURL   string
	AppKey    string
	AppSecret string
	TrID      string
	CustType  string
	Manager   *TokenManager
}

// FetchBasicInfo calls CTPF1002R (or env-configured tr_id) to look up stock metadata.
func (c *DomesticInfoClient) FetchBasicInfo(prdtTypeCd, pdno string) (DomesticStockInfo, error) {
	token, err := c.Manager.GetToken()
	if err != nil {
		return DomesticStockInfo{}, err
	}
	body, err := GetWithRetry(
		c.HTTP,
		c.BaseURL+"/uapi/domestic-stock/v1/quotations/search-stock-info",
		map[string]string{
			"PRDT_TYPE_CD": prdtTypeCd,
			"PDNO":         pdno,
		},
		BuildHeaders(token, c.AppKey, c.AppSecret, c.TrID, c.CustType),
		c.Manager, c.AppKey, c.AppSecret, c.TrID, c.CustType,
	)
	if err != nil {
		return DomesticStockInfo{}, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return DomesticStockInfo{}, err
	}
	output, err := extractOutputObject(raw)
	if err != nil {
		return DomesticStockInfo{}, fmt.Errorf("KIS domestic info: %w", err)
	}

	name := output["prdt_name"]
	if name == "" {
		name = output["prdt_name120"]
	}
	if name == "" {
		name = output["prdt_abrv_name"]
	}
	if name == "" {
		name = output["prdt_eng_name"]
	}

	return DomesticStockInfo{
		Pdno:        output["pdno"],
		PrdtTypeCd:  output["prdt_type_cd"],
		MarketID:    output["mket_id_cd"],
		Name:        name,
		SctyGrpIDCd: output["scty_grp_id_cd"],
		EtfDvsnCd:   output["etf_dvsn_cd"],
	}, nil
}

// ClassifyAssetClass looks up a domestic ticker and returns "etf" or "stock".
// prdtTypeCd "300" covers 주식/ETF/ETN/ELW.
func (c *DomesticInfoClient) ClassifyAssetClass(ticker string) (string, error) {
	info, err := c.FetchBasicInfo("300", ticker)
	if err != nil {
		return "", err
	}
	return info.AssetClass(), nil
}

func extractOutputObject(raw map[string]json.RawMessage) (map[string]string, error) {
	blob, ok := raw["output"]
	if !ok {
		return nil, fmt.Errorf("missing output field")
	}
	var arr []map[string]string
	if json.Unmarshal(blob, &arr) == nil && len(arr) > 0 {
		return arr[0], nil
	}
	var obj map[string]string
	if json.Unmarshal(blob, &obj) == nil && len(obj) > 0 {
		return obj, nil
	}
	return nil, fmt.Errorf("empty or unreadable output")
}
