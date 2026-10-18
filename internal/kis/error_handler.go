package kis

import "encoding/json"

type responseStatus struct {
	MsgCd string `json:"msg_cd"`
}

// IsTokenExpiredError reports whether a KIS response indicates token expiry.
// KIS returns HTTP 500 with msg_cd "EGW00123" when the OAuth token has expired.
func IsTokenExpiredError(statusCode int, body []byte) bool {
	return msgCode(statusCode, body) == "EGW00123"
}

// IsRateLimitError reports whether a KIS response indicates the per-second
// request limit was exceeded. KIS returns HTTP 500 with msg_cd "EGW00201"
// ("초당 거래건수를 초과하였습니다") and rejects the request without processing it.
func IsRateLimitError(statusCode int, body []byte) bool {
	return msgCode(statusCode, body) == "EGW00201"
}

// msgCode extracts msg_cd from a KIS HTTP 500 error body, or "" otherwise.
func msgCode(statusCode int, body []byte) string {
	if statusCode != 500 {
		return ""
	}
	var meta responseStatus
	if err := json.Unmarshal(body, &meta); err != nil {
		return ""
	}
	return meta.MsgCd
}
