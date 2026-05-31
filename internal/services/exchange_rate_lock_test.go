package services

import (
	"sync"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

type blockingEximClient struct {
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (c *blockingEximClient) FetchUSDRate(string) (float64, error) {
	c.once.Do(func() { close(c.entered) })
	<-c.release
	return 1320, nil
}

func TestExchangeRateServiceDoesNotHoldLockDuringFetch(t *testing.T) {
	client := &blockingEximClient{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	svc := NewEximExchangeRateService(client)

	done := make(chan struct{})
	go func() {
		_ = svc.GetUSDKRW()
		close(done)
	}()

	select {
	case <-client.entered:
	case <-time.After(time.Second):
		close(client.release)
		t.Fatal("FetchUSDRate was not called")
	}

	lockAcquired := make(chan struct{})
	go func() {
		svc.mu.Lock()
		svc.cachedRates["20990101"] = numeric.Zero
		svc.mu.Unlock()
		close(lockAcquired)
	}()

	select {
	case <-lockAcquired:
	case <-time.After(100 * time.Millisecond):
		close(client.release)
		t.Fatal("service lock is held during FetchUSDRate")
	}

	close(client.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GetUSDKRW did not return after fetch release")
	}
}
