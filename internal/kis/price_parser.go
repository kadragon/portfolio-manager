package kis

import (
	"encoding/json"
	"strconv"
	"strings"
)

// KisPriceQuote is a parsed price result from KIS API.
type KisPriceQuote struct {
	Symbol   string
	Name     string
	Price    float64
	Currency string
	Exchange string // price-form code ("NASD", "NYSE", "AMEX") or empty for domestic
}

// ParseKoreaPrice parses a KIS domestic-stock inquire-price response.
// symbol is the fallback ticker when the response lacks stck_code.
func ParseKoreaPrice(data []byte, symbol string) KisPriceQuote {
	output := extractFirstOutput(data)
	resolvedSymbol := output["stck_code"]
	if resolvedSymbol == "" {
		resolvedSymbol = symbol
	}
	return KisPriceQuote{
		Symbol:   resolvedSymbol,
		Name:     output["hts_kor_isnm"],
		Price:    parseFloat(output["stck_prpr"]),
		Currency: "KRW",
	}
}

// usNameKeys mirrors the Python parse_us_price fallback order.
var usNameKeys = []string{
	"name", "enname", "ename", "en_name", "symb_name",
	"symbol_name", "prdt_name", "product_name", "item_name",
}

// ParseUSPrice parses a KIS overseas-stock inquire-price response.
func ParseUSPrice(data []byte, symbol, exchange string) KisPriceQuote {
	output := extractFirstOutput(data)

	name := ""
	for _, key := range usNameKeys {
		if v := strings.TrimSpace(output[key]); v != "" {
			name = v
			break
		}
	}

	resolvedSymbol := output["symbol"]
	if resolvedSymbol == "" {
		resolvedSymbol = output["symb"]
	}
	if resolvedSymbol == "" {
		resolvedSymbol = output["rsym"]
	}
	if resolvedSymbol == "" {
		resolvedSymbol = symbol
	}

	// Use "last" (real-time); fall back to "base" (previous close) when market is closed.
	// KIS returns last="0" or absent outside trading hours; base holds the prior session close.
	priceStr := strings.TrimSpace(output["last"])
	if priceStr == "" || priceStr == "0" {
		priceStr = strings.TrimSpace(output["base"])
	}

	return KisPriceQuote{
		Symbol:   resolvedSymbol,
		Name:     name,
		Price:    parseFloat(priceStr),
		Currency: "USD",
		Exchange: exchange,
	}
}

// extractFirstOutput handles KIS responses where "output" is either a JSON object
// or a JSON array (takes index 0). Returns map[string]string for field access.
func extractFirstOutput(data []byte) map[string]string {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return map[string]string{}
	}
	rawOutput, ok := raw["output"]
	if !ok {
		return map[string]string{}
	}

	// Try array first
	var arr []map[string]string
	if json.Unmarshal(rawOutput, &arr) == nil && len(arr) > 0 {
		return arr[0]
	}

	// Try object
	var obj map[string]string
	if json.Unmarshal(rawOutput, &obj) == nil {
		return obj
	}
	return map[string]string{}
}

// ParseKISStatus reads the top-level rt_cd/msg_cd/msg1 from a KIS response.
// Returns rtCd, msgCd, msg1. On unmarshal failure all are empty.
func ParseKISStatus(data []byte) (rtCd, msgCd, msg1 string) {
	var meta struct {
		RtCd  string `json:"rt_cd"`
		MsgCd string `json:"msg_cd"`
		Msg1  string `json:"msg1"`
	}
	_ = json.Unmarshal(data, &meta)
	return meta.RtCd, meta.MsgCd, meta.Msg1
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
