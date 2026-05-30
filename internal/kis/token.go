package kis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kadragon/portfolio-manager/internal/ktime"
)

// TokenData holds a KIS OAuth token and its expiry.
type TokenData struct {
	Token     string
	ExpiresAt time.Time
}

// TokenStore persists KIS OAuth tokens across requests.
type TokenStore interface {
	Save(token string, expiresAt time.Time) error
	Load() (*TokenData, error)
}

// MemoryTokenStore is a non-persistent in-memory store (test / single-binary use).
type MemoryTokenStore struct {
	data *TokenData
}

func (s *MemoryTokenStore) Save(token string, expiresAt time.Time) error {
	s.data = &TokenData{Token: token, ExpiresAt: expiresAt}
	return nil
}

func (s *MemoryTokenStore) Load() (*TokenData, error) {
	return s.data, nil
}

type fileTokenPayload struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// FileTokenStore persists the token to a JSON file, byte-compatible with
// the Python FileTokenStore at .data/kis_token_{n}.json.
type FileTokenStore struct {
	path string
}

// NewFileTokenStore creates a FileTokenStore writing to the given path.
func NewFileTokenStore(path string) *FileTokenStore {
	return &FileTokenStore{path: path}
}

func (s *FileTokenStore) Save(token string, expiresAt time.Time) error {
	payload := fileTokenPayload{
		Token:     token,
		ExpiresAt: expiresAt.In(ktime.KST).Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *FileTokenStore) Load() (*TokenData, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var payload fileTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	t, err := parseTokenTime(payload.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &TokenData{Token: payload.Token, ExpiresAt: t}, nil
}

// parseTokenTime parses ISO-8601 / Python isoformat strings produced by both
// the Go and Python implementations (RFC3339 with offset, or naive KST).
var tokenTimeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
}

func parseTokenTime(s string) (time.Time, error) {
	for i, layout := range tokenTimeLayouts {
		if i == 0 {
			if t, err := time.Parse(layout, s); err == nil {
				if t.Location() == time.UTC {
					return t.In(ktime.KST), nil
				}
				return t, nil
			}
		} else {
			if t, err := time.ParseInLocation(layout, s, ktime.KST); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("kis: cannot parse token time %q", s)
}
