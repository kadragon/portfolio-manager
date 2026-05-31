package kis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadragon/portfolio-manager/internal/ktime"
)

// AuthClient requests KIS OAuth tokens via POST /oauth2/tokenP.
type AuthClient struct {
	HTTPClient *http.Client
	BaseURL    string
	AppKey     string
	AppSecret  string
}

type authRequest struct {
	GrantType string `json:"grant_type"`
	AppKey    string `json:"appkey"`
	AppSecret string `json:"appsecret"`
}

type authResponse struct {
	AccessToken             string `json:"access_token"`
	ExpiresIn               int    `json:"expires_in"`
	AccessTokenTokenExpired string `json:"access_token_token_expired"`
}

// RequestAccessToken obtains a fresh KIS OAuth token.
func (c *AuthClient) RequestAccessToken() (TokenData, error) {
	body, err := json.Marshal(authRequest{
		GrantType: "client_credentials",
		AppKey:    c.AppKey,
		AppSecret: c.AppSecret,
	})
	if err != nil {
		return TokenData{}, err
	}
	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/oauth2/tokenP",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return TokenData{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenData{}, err
	}
	if resp.StatusCode >= 400 {
		return TokenData{}, fmt.Errorf("KIS auth HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	var ar authResponse
	if err := json.Unmarshal(respBody, &ar); err != nil {
		return TokenData{}, err
	}

	var expiresAt time.Time
	if ar.ExpiresIn > 0 {
		expiresAt = ktime.NowKST().Add(time.Duration(ar.ExpiresIn) * time.Second)
	} else if ar.AccessTokenTokenExpired != "" {
		// Python stores KST-naive datetime like "2026-01-03 13:21:44"
		expiresAt, err = time.ParseInLocation("2006-01-02 15:04:05", ar.AccessTokenTokenExpired, ktime.KST)
		if err != nil {
			return TokenData{}, fmt.Errorf("KIS auth: parse expires %q: %w", ar.AccessTokenTokenExpired, err)
		}
	}
	return TokenData{Token: ar.AccessToken, ExpiresAt: expiresAt}, nil
}
