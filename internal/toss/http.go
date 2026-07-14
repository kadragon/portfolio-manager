package toss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// apiEnvelope decodes Toss's uniform {"result": T} success envelope.
type apiEnvelope[T any] struct {
	Result T `json:"result"`
}

type apiErrorResponse struct {
	Error struct {
		RequestID string `json:"requestId"`
		Code      string `json:"code"`
		Message   string `json:"message"`
	} `json:"error"`
}

// doGet issues an authenticated GET against path, sets query params (empty
// values are skipped so callers can pass optional filters uniformly) and any
// extra headers (e.g. X-Tossinvest-Account), and decodes a successful
// {"result": T} envelope. prefix labels errors (e.g. "toss orderbook").
func doGet[T any](ctx context.Context, c *Client, prefix, path string, query, headers map[string]string) (T, error) {
	var zero T
	token, err := c.accessToken()
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return zero, fmt.Errorf("%s: create request: %w", prefix, err)
	}
	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		req.URL.RawQuery = q.Encode()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doRequest[T](c, prefix, req)
}

// doPost issues an authenticated POST with a JSON body and decodes the
// {"result": T} envelope. Pass a nil body for endpoints with no request body
// (e.g. cancelOrder).
func doPost[T any](ctx context.Context, c *Client, prefix, path string, body any, headers map[string]string) (T, error) {
	var zero T
	token, err := c.accessToken()
	if err != nil {
		return zero, err
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("%s: json marshal: %w", prefix, err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, reader) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return zero, fmt.Errorf("%s: create request: %w", prefix, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doRequest[T](c, prefix, req)
}

// doDelete issues an authenticated DELETE and decodes the {"result": T}
// envelope.
func doDelete[T any](ctx context.Context, c *Client, prefix, path string, headers map[string]string) (T, error) {
	var zero T
	token, err := c.accessToken()
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+path, nil) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return zero, fmt.Errorf("%s: create request: %w", prefix, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doRequest[T](c, prefix, req)
}

func doRequest[T any](c *Client, prefix string, req *http.Request) (T, error) {
	var zero T
	resp, err := c.HTTP.Do(req) //nolint:gosec // BaseURL is operator-controlled config or httptest URL.
	if err != nil {
		return zero, fmt.Errorf("%s: request: %w", prefix, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("%s: read response: %w", prefix, err)
	}
	if resp.StatusCode >= 400 {
		return zero, parseAPIError(prefix, resp.StatusCode, respBody)
	}
	if resp.StatusCode == http.StatusNoContent {
		// cancelConditionalOrder is the one documented 204 No Content
		// endpoint. Gate on the status code, not "body happens to be
		// empty" — a truncated/malformed 200 response must still fail
		// json.Unmarshal below rather than silently report a zero-value
		// result (e.g. empty holdings, zero balance) as success.
		return zero, nil
	}
	var env apiEnvelope[T]
	if err := json.Unmarshal(respBody, &env); err != nil {
		return zero, fmt.Errorf("%s: json unmarshal: %w", prefix, err)
	}
	return env.Result, nil
}

func parseAPIError(prefix string, status int, body []byte) error {
	var data apiErrorResponse
	if err := json.Unmarshal(body, &data); err == nil && data.Error.Code != "" {
		if data.Error.RequestID != "" {
			return fmt.Errorf("%s HTTP %d: %s: %s (request_id=%s)",
				prefix, status, data.Error.Code, data.Error.Message, data.Error.RequestID)
		}
		return fmt.Errorf("%s HTTP %d: %s: %s", prefix, status, data.Error.Code, data.Error.Message)
	}
	return fmt.Errorf("%s HTTP %d: %s", prefix, status, string(body))
}

func parseOAuthError(status int, body []byte) error {
	var data struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &data); err == nil && data.Error != "" {
		if data.ErrorDescription != "" {
			return fmt.Errorf("toss auth HTTP %d: %s: %s", status, data.Error, data.ErrorDescription)
		}
		return fmt.Errorf("toss auth HTTP %d: %s", status, data.Error)
	}
	return fmt.Errorf("toss auth HTTP %d: %s", status, string(body))
}
