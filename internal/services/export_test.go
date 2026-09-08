package services

import (
	"testing"
	"time"
)

// SetSyncCallDelayForTest shortens the inter-call pacing for the duration of one
// test and restores it afterwards. Price sync spends almost all of its wall clock
// in syncCallDelay; a test exercising a multi-day walk-back would otherwise pay
// seconds per case for rate limiting no fake client needs.
func SetSyncCallDelayForTest(t *testing.T, d time.Duration) {
	t.Helper()
	prev := syncCallDelay
	syncCallDelay = d
	t.Cleanup(func() { syncCallDelay = prev })
}
