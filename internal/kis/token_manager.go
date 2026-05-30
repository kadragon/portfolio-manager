package kis

import (
	"sync"
	"time"

	"github.com/kadragon/portfolio-manager/internal/ktime"
)

const defaultSkew = time.Minute

// TokenManager caches a KIS OAuth token and refreshes it before expiry.
// Safe for concurrent use.
type TokenManager struct {
	mu    sync.Mutex
	store TokenStore
	auth  *AuthClient
	skew  time.Duration
}

// NewTokenManager creates a TokenManager with the given token store and auth client.
// skew ≤ 0 defaults to 1 minute.
func NewTokenManager(store TokenStore, auth *AuthClient, skew time.Duration) *TokenManager {
	if skew <= 0 {
		skew = defaultSkew
	}
	return &TokenManager{store: store, auth: auth, skew: skew}
}

// GetToken returns a valid token, refreshing from the auth client when the cached
// token is within skew of expiry.
func (m *TokenManager) GetToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cached, _ := m.store.Load()
	now := ktime.NowKST()
	if cached != nil && cached.ExpiresAt.After(now.Add(m.skew)) {
		return cached.Token, nil
	}
	return m.refresh()
}

// RefreshToken forces a new token regardless of the cached expiry.
// Used after a KIS server-side token expiry error (500 + EGW00123).
func (m *TokenManager) RefreshToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.refresh()
}

func (m *TokenManager) refresh() (string, error) {
	td, err := m.auth.RequestAccessToken()
	if err != nil {
		return "", err
	}
	_ = m.store.Save(td.Token, td.ExpiresAt)
	return td.Token, nil
}
