package kis

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DomesticStockInfo holds basic info for a KOSPI/KOSDAQ stock.
type DomesticStockInfo struct {
	Pdno       string
	PrdtTypeCd string
	MarketID   string
	Name       string
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
		Pdno:       output["pdno"],
		PrdtTypeCd: output["prdt_type_cd"],
		MarketID:   output["mket_id_cd"],
		Name:       name,
	}, nil
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
