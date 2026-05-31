package container

import (
	"reflect"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/kis"
)

func setKISEnv(t *testing.T) {
	t.Helper()
	t.Setenv("KIS_APP_KEY", "app-key")
	t.Setenv("KIS_APP_SECRET", "app-secret")
	t.Setenv("KIS_ENV", "demo")
	t.Setenv("KIS_CANO", "12345678")
	t.Setenv("KIS_ACNT_PRDT_CD", "01")
	t.Setenv("KIS_CUST_TYPE", "P")
}

func TestKISClientsShareTokenManager(t *testing.T) {
	setKISEnv(t)

	auth := buildKISAuth()
	if auth == nil {
		t.Fatal("expected KIS auth config")
	}
	priceClient, ok := buildKISClient(auth).(*kis.UnifiedPriceClient)
	if !ok {
		t.Fatal("expected unified price client")
	}
	orderClient, ok := buildOrderClient(auth).(*kis.UnifiedOrderClient)
	if !ok {
		t.Fatal("expected unified order client")
	}
	balanceClient, ok := buildBalanceClient(auth).(*kis.DomesticBalanceClient)
	if !ok {
		t.Fatal("expected domestic balance client")
	}

	manager := priceClient.Domestic.Manager
	if manager == nil {
		t.Fatal("price manager is nil")
	}
	if priceClient.Overseas.Manager != manager ||
		priceClient.DomesticInfo.Manager != manager ||
		priceClient.OverseasInfo.Manager != manager ||
		orderClient.Domestic.Manager != manager ||
		orderClient.Overseas.Manager != manager ||
		balanceClient.Manager != manager {
		t.Fatal("KIS clients do not share one token manager")
	}
}

func TestContainerWiresAccountSyncIntoRebalanceExecution(t *testing.T) {
	setKISEnv(t)
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	c := newWithQueries(sqlDB, q, true)
	if c.AccountSync == nil {
		t.Fatal("account sync is not configured")
	}
	field := reflect.ValueOf(c.RebalanceExecution).Elem().FieldByName("syncService")
	if field.IsNil() {
		t.Fatal("rebalance execution sync service is nil")
	}
}
