package kis

import "encoding/json"

type responseStatus struct {
	MsgCd string `json:"msg_cd"`
}

// IsTokenExpiredError reports whether a KIS response indicates token expiry.
// KIS returns HTTP 500 with msg_cd "EGW00123" when the OAuth token has expired.
func IsTokenExpiredError(statusCode int, body []byte) bool {
	if statusCode != 500 {
		return false
	}
	var meta responseStatus
	if err := json.Unmarshal(body, &meta); err != nil {
		return false
	}
	return meta.MsgCd == "EGW00123"
}
