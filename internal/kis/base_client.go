package kis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BuildHeaders constructs KIS API request headers for a GET request.
func BuildHeaders(token, appKey, appSecret, trID, custType string) map[string]string {
	return map[string]string{
		"content-type":  "application/json",
		"authorization": "Bearer " + token,
		"appkey":        appKey,
		"appsecret":     appSecret,
		"tr_id":         trID,
		"custtype":      custType,
	}
}

// TrIDForEnv selects the real or demo TR-ID based on KIS_ENV value.
func TrIDForEnv(env, realID, demoID string) (string, error) {
	env = strings.ToLower(strings.TrimSpace(env))
	if i := strings.IndexByte(env, '/'); i >= 0 {
		env = env[:i]
	}
	switch env {
	case "real", "prod":
		return realID, nil
	case "demo", "vps", "paper":
		return demoID, nil
	}
	return "", fmt.Errorf("kis: env must be real/prod or demo/vps/paper, got %q", env)
}

// GetWithRetry issues a GET request, retrying once with a refreshed token on EGW00123.
// headers must be the initial request headers; on retry they are rebuilt from appKey/appSecret/trID/custType.
func GetWithRetry(
	client *http.Client,
	url string,
	params, headers map[string]string,
	manager *TokenManager,
	appKey, appSecret, trID, custType string,
) ([]byte, error) {
	body, status, err := doGet(client, url, params, headers)
	if err != nil {
		return nil, err
	}
	if IsTokenExpiredError(status, body) && manager != nil {
		newToken, refreshErr := manager.RefreshToken()
		if refreshErr != nil {
			return nil, fmt.Errorf("KIS token refresh: %w", refreshErr)
		}
		body, status, err = doGet(client, url, params, BuildHeaders(newToken, appKey, appSecret, trID, custType))
		if err != nil {
			return nil, err
		}
	}
	if status >= 400 {
		return nil, fmt.Errorf("KIS HTTP %d: %s", status, string(body))
	}
	return body, nil
}

// postWithRetry issues a POST request with a JSON body, retrying once on EGW00123.
func postWithRetry(
	client *http.Client,
	url string,
	payload any,
	headers map[string]string,
	manager *TokenManager,
	appKey, appSecret, trID, custType string,
) ([]byte, error) {
	body, status, err := doPost(client, url, payload, headers)
	if err != nil {
		return nil, err
	}
	if IsTokenExpiredError(status, body) && manager != nil {
		newToken, refreshErr := manager.RefreshToken()
		if refreshErr != nil {
			return nil, fmt.Errorf("KIS token refresh: %w", refreshErr)
		}
		body, status, err = doPost(client, url, payload, BuildHeaders(newToken, appKey, appSecret, trID, custType))
		if err != nil {
			return nil, err
		}
	}
	if status >= 400 {
		return nil, fmt.Errorf("KIS HTTP %d: %s", status, string(body))
	}
	return body, nil
}

func doPost(client *http.Client, url string, payload any, headers map[string]string) ([]byte, int, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// GetWithRetryFull is like GetWithRetry but also returns the response headers.
// Used by paginated endpoints that need response metadata (e.g. tr_cont for balance pagination).
func GetWithRetryFull(
	client *http.Client,
	url string,
	params, headers map[string]string,
	manager *TokenManager,
	appKey, appSecret, trID, custType string,
) ([]byte, http.Header, error) {
	body, status, respHeaders, err := doGetFull(client, url, params, headers)
	if err != nil {
		return nil, nil, err
	}
	if IsTokenExpiredError(status, body) && manager != nil {
		newToken, refreshErr := manager.RefreshToken()
		if refreshErr != nil {
			return nil, nil, fmt.Errorf("KIS token refresh: %w", refreshErr)
		}
		body, status, respHeaders, err = doGetFull(client, url, params, BuildHeaders(newToken, appKey, appSecret, trID, custType))
		if err != nil {
			return nil, nil, err
		}
	}
	if status >= 400 {
		return nil, nil, fmt.Errorf("KIS HTTP %d: %s", status, string(body))
	}
	return body, respHeaders, nil
}

func doGetFull(client *http.Client, url string, params, headers map[string]string) ([]byte, int, http.Header, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, resp.Header, err
}

func doGet(client *http.Client, url string, params, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}
