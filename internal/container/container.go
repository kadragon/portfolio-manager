// Package container wires the repositories (and, in later phases, services and
// external clients) over an open database, the Go counterpart of
// core/container.py's ServiceContainer. As the composition root it may depend on
// both the db and repository layers.
package container

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/kis"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// Container holds shared dependencies for the web layer.
type Container struct {
	DB                  *sql.DB
	Groups              *repositories.GroupRepository
	Stocks              *repositories.StockRepository
	Accounts            *repositories.AccountRepository
	Holdings            *repositories.HoldingRepository
	Deposits            *repositories.DepositRepository
	StockPrices         *repositories.StockPriceRepository
	OrderExecutions     *repositories.OrderExecutionRepository
	Portfolio           *services.PortfolioService
	Rebalance           *services.RebalanceService
	RebalanceExecution  *services.RebalanceExecutionService
	OrderClient         services.OrderClient                      // nil if KIS not configured
	AccountSync         *services.KisAccountSyncService           // nil if KIS not configured; key-1 service
	AccountSyncByKeyID  map[int64]*services.KisAccountSyncService // keyed by kis_api_key_id
	PriceSync           *services.PriceSyncService                // nil if KIS not configured
	StockClassification *services.StockClassificationService      // backfills asset_class via KIS; Enabled()==false if KIS absent
	KisCano             string
	KisAcntPrdtCd       string
}

// New opens the database at path (empty = default location) and builds the
// repositories. The caller is responsible for Close.
func New(path string) (*Container, error) {
	sqlDB, q, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	return newWithQueries(sqlDB, q, true), nil
}

// NewWithQueries builds a Container over an already-open database and queries
// handle (used by tests with an in-memory database; skips KIS wiring).
func NewWithQueries(sqlDB *sql.DB, q *sqlc.Queries) *Container {
	return newWithQueries(sqlDB, q, false)
}

func newWithQueries(sqlDB *sql.DB, q *sqlc.Queries, setupKIS bool) *Container {
	groups := repositories.NewGroupRepository(q)
	stocks := repositories.NewStockRepository(q)
	accounts := repositories.NewAccountRepository(q)
	holdings := repositories.NewHoldingRepository(q, sqlDB)
	deposits := repositories.NewDepositRepository(q)
	stockPrices := repositories.NewStockPriceRepository(q)
	orderExecutions := repositories.NewOrderExecutionRepository(q)

	var priceClient services.PriceClient
	var exchangeRate *services.ExchangeRateService
	var orderClient services.OrderClient
	var accountSync *services.KisAccountSyncService
	accountSyncByKeyID := map[int64]*services.KisAccountSyncService{}
	var rebalanceSync services.SyncService
	var assetClassifier services.AssetClassifier
	kisCano := ""
	kisAcntPrdtCd := ""

	if setupKIS {
		kisAuth := buildKISAuth()
		priceClient = buildKISClient(kisAuth)
		exchangeRate = buildExchangeRate()
		orderClient = buildOrderClient(kisAuth)
		assetClassifier = buildAssetClassifier(kisAuth)

		kisCano, kisAcntPrdtCd = loadKISAccount()
		if balanceClient := buildBalanceClient(kisAuth); balanceClient != nil {
			accountSync = services.NewKisAccountSyncService(accounts, holdings, stocks, groups, balanceClient, ".data/kis_sync.log")
			accountSync.SetClassifier(assetClassifier)
			accountSyncByKeyID[1] = accountSync
		}

		if kisAuth != nil {
			for id := 2; id <= 9; id++ {
				if auth := buildKISAuthExtra(id, kisAuth); auth != nil {
					bc := buildBalanceClientFromAuth(auth)
					svc := services.NewKisAccountSyncService(accounts, holdings, stocks, groups, bc,
						fmt.Sprintf(".data/kis_sync_%d.log", id))
					svc.SetClassifier(buildAssetClassifier(auth))
					accountSyncByKeyID[int64(id)] = svc
				}
			}
		}

		if accountSync != nil || len(accountSyncByKeyID) > 0 {
			rebalanceSync = &rebalanceSyncAdapter{
				accounts:    accounts,
				sync:        accountSync,
				syncByKeyID: accountSyncByKeyID,
				cano:        kisCano,
				acntPrdtCd:  kisAcntPrdtCd,
			}
		}
	}

	priceService := services.NewPriceService(stockPrices)
	_ = services.NewStockService(stocks, priceService)
	portfolio := services.NewPortfolioService(groups, stocks, holdings, accounts, deposits, priceService, exchangeRate)
	rebalance := services.NewRebalanceService()

	var priceSync *services.PriceSyncService
	if priceClient != nil {
		priceSync = services.NewPriceSyncService(priceClient, stockPrices, stocks)
	}

	execRepo := &execRepoAdapter{r: orderExecutions}
	rebalanceExecution := services.NewRebalanceExecutionService(orderClient, execRepo, rebalanceSync)
	stockClassification := services.NewStockClassificationService(stocks, assetClassifier)

	return &Container{
		DB:                  sqlDB,
		Groups:              groups,
		Stocks:              stocks,
		Accounts:            accounts,
		Holdings:            holdings,
		Deposits:            deposits,
		StockPrices:         stockPrices,
		OrderExecutions:     orderExecutions,
		Portfolio:           portfolio,
		Rebalance:           rebalance,
		RebalanceExecution:  rebalanceExecution,
		OrderClient:         orderClient,
		AccountSync:         accountSync,
		AccountSyncByKeyID:  accountSyncByKeyID,
		PriceSync:           priceSync,
		StockClassification: stockClassification,
		KisCano:             kisCano,
		KisAcntPrdtCd:       kisAcntPrdtCd,
	}
}

// Close releases the database connection.
func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// execRepoAdapter wraps OrderExecutionRepository to satisfy services.ExecutionRepo.
type execRepoAdapter struct {
	r *repositories.OrderExecutionRepository
}

func (a *execRepoAdapter) Create(
	ctx context.Context,
	ticker, side string,
	quantity int,
	currency, status, message, exchange string,
	rawResponse map[string]any,
) (models.OrderExecutionRecord, error) {
	return a.r.Create(ctx, ticker, side, quantity, currency, status, message, exchange, rawResponse)
}

type rebalanceSyncAdapter struct {
	accounts    *repositories.AccountRepository
	sync        *services.KisAccountSyncService
	syncByKeyID map[int64]*services.KisAccountSyncService
	cano        string
	acntPrdtCd  string
}

func (a *rebalanceSyncAdapter) SyncAccount() error {
	ctx := context.Background()
	accounts, err := a.accounts.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		svc := resolveSyncService(a.sync, a.syncByKeyID, account.KisAPIKeyID)
		if svc == nil {
			continue
		}

		cano, acntPrdtCd := a.cano, a.acntPrdtCd
		if account.KisAccountNo != nil && *account.KisAccountNo != "" {
			parsedCano, parsedPrdtCd := parseKISAccountNo(*account.KisAccountNo)
			if parsedCano == "" {
				continue
			}
			cano, acntPrdtCd = parsedCano, parsedPrdtCd
		}
		if cano == "" {
			continue
		}
		if _, err := svc.SyncAccount(ctx, account, cano, acntPrdtCd, false); err != nil {
			return err
		}
	}
	return nil
}

// SyncServiceForKeyID returns the KisAccountSyncService for the given kis_api_key_id,
// falling back to AccountSync (key-1). Logs a warning when keyID is set but not found.
func (c *Container) SyncServiceForKeyID(keyID *int64) *services.KisAccountSyncService {
	return resolveSyncService(c.AccountSync, c.AccountSyncByKeyID, keyID)
}

func resolveSyncService(
	defaultSync *services.KisAccountSyncService,
	byKeyID map[int64]*services.KisAccountSyncService,
	keyID *int64,
) *services.KisAccountSyncService {
	if keyID != nil {
		if s, ok := byKeyID[*keyID]; ok {
			return s
		}
		if *keyID != 1 {
			log.Printf("warn: no sync service for requested KIS key, falling back to key-1")
		}
	}
	return defaultSync
}

func parseKISAccountNo(raw string) (cano, acntPrdtCd string) {
	var digits strings.Builder
	for _, ch := range raw {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}
	if d := digits.String(); len(d) == 10 {
		return d[:8], d[8:]
	}
	return "", ""
}

type kisAuth struct {
	appKey       string
	appSecret    string
	env          string
	custType     string
	baseURL      string
	cano         string
	acntPrdtCd   string
	httpClient   *http.Client
	tokenManager *kis.TokenManager
}

func buildKISAuth() *kisAuth {
	appKey := strings.TrimSpace(os.Getenv("KIS_APP_KEY"))
	appSecret := strings.TrimSpace(os.Getenv("KIS_APP_SECRET"))
	if appKey == "" || appSecret == "" {
		return nil
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("KIS_ENV")))
	if env == "" {
		env = "real"
	}
	if i := strings.IndexByte(env, '/'); i >= 0 {
		env = env[:i]
	}

	custType := strings.TrimSpace(os.Getenv("KIS_CUST_TYPE"))
	if custType == "" {
		custType = "P"
	}

	baseURL := "https://openapi.koreainvestment.com:9443"
	if env == "demo" || env == "vps" || env == "paper" {
		baseURL = "https://openapivts.koreainvestment.com:29443"
	}

	cano, acntPrdtCd := loadKISAccount()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	authClient := &kis.AuthClient{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
		AppKey:     appKey,
		AppSecret:  appSecret,
	}
	store := kis.NewFileTokenStore(".data/kis_token_1.json")
	manager := kis.NewTokenManager(store, authClient, time.Minute)

	return &kisAuth{
		appKey:       appKey,
		appSecret:    appSecret,
		env:          env,
		custType:     custType,
		baseURL:      baseURL,
		cano:         cano,
		acntPrdtCd:   acntPrdtCd,
		httpClient:   httpClient,
		tokenManager: manager,
	}
}

// buildKISClient returns a UnifiedPriceClient, or nil if KIS keys are absent.
func buildKISClient(auth *kisAuth) services.PriceClient {
	if auth == nil {
		return nil
	}

	prdtTypeCd := strings.TrimSpace(os.Getenv("KIS_PRDT_TYPE_CD"))
	if prdtTypeCd == "" {
		prdtTypeCd = "300"
	}
	domesticInfoTrID := strings.TrimSpace(os.Getenv("KIS_DOMESTIC_INFO_TR_ID"))
	if domesticInfoTrID == "" {
		domesticInfoTrID = "CTPF1002R"
	}
	overseasInfoTrID := strings.TrimSpace(os.Getenv("KIS_OVERSEAS_INFO_TR_ID"))
	if overseasInfoTrID == "" {
		overseasInfoTrID = "CTPF1702R"
	}

	unified := &kis.UnifiedPriceClient{
		Domestic: &kis.DomesticPriceClient{
			HTTP:      auth.httpClient,
			BaseURL:   auth.baseURL,
			AppKey:    auth.appKey,
			AppSecret: auth.appSecret,
			CustType:  auth.custType,
			Env:       auth.env,
			Manager:   auth.tokenManager,
		},
		Overseas: &kis.OverseasPriceClient{
			HTTP:      auth.httpClient,
			BaseURL:   auth.baseURL,
			AppKey:    auth.appKey,
			AppSecret: auth.appSecret,
			CustType:  auth.custType,
			Env:       auth.env,
			Manager:   auth.tokenManager,
		},
		DomesticInfo: &kis.DomesticInfoClient{
			HTTP:      auth.httpClient,
			BaseURL:   auth.baseURL,
			AppKey:    auth.appKey,
			AppSecret: auth.appSecret,
			TrID:      domesticInfoTrID,
			CustType:  auth.custType,
			Manager:   auth.tokenManager,
		},
		OverseasInfo: &kis.OverseasInfoClient{
			HTTP:      auth.httpClient,
			BaseURL:   auth.baseURL,
			AppKey:    auth.appKey,
			AppSecret: auth.appSecret,
			TrID:      overseasInfoTrID,
			CustType:  auth.custType,
			Manager:   auth.tokenManager,
		},
		PrdtTypeCd: prdtTypeCd,
	}

	log.Printf("KIS price client initialized (env=%q)", auth.env) //nolint:gosec // env is operator-controlled, not user input
	return unified
}

// loadKISAccount resolves the KIS account number from env vars.
// Prefers explicit KIS_CANO + KIS_ACNT_PRDT_CD; falls back to KIS_ACCOUNT_NO
// (10 consecutive digits split 8+2), matching Python ServiceContainer behaviour.
func loadKISAccount() (cano, acntPrdtCd string) {
	cano = strings.TrimSpace(os.Getenv("KIS_CANO"))
	acntPrdtCd = strings.TrimSpace(os.Getenv("KIS_ACNT_PRDT_CD"))
	if cano != "" && acntPrdtCd != "" {
		return
	}
	if raw := strings.TrimSpace(os.Getenv("KIS_ACCOUNT_NO")); raw != "" {
		var digits strings.Builder
		for _, ch := range raw {
			if ch >= '0' && ch <= '9' {
				digits.WriteRune(ch)
			}
		}
		if d := digits.String(); len(d) == 10 {
			cano = d[:8]
			acntPrdtCd = d[8:]
		}
	}
	if acntPrdtCd == "" {
		acntPrdtCd = "01"
	}
	return
}

// buildOrderClient returns a UnifiedOrderClient, or nil if keys/account are absent.
func buildOrderClient(auth *kisAuth) services.OrderClient {
	if auth == nil || auth.cano == "" {
		return nil
	}

	log.Printf("KIS order client initialized (env=%q)", auth.env) //nolint:gosec // env is operator-controlled, not user input
	return &kis.UnifiedOrderClient{
		Domestic: &kis.DomesticOrderClient{
			HTTP:       auth.httpClient,
			BaseURL:    auth.baseURL,
			AppKey:     auth.appKey,
			AppSecret:  auth.appSecret,
			CANO:       auth.cano,
			AcntPrdtCd: auth.acntPrdtCd,
			CustType:   auth.custType,
			Env:        auth.env,
			Manager:    auth.tokenManager,
		},
		Overseas: &kis.OverseasOrderClient{
			HTTP:      auth.httpClient,
			BaseURL:   auth.baseURL,
			AppKey:    auth.appKey,
			AppSecret: auth.appSecret,
			CustType:  auth.custType,
			Env:       auth.env,
			Manager:   auth.tokenManager,
		},
	}
}

// domesticInfoClassifier / overseasInfoClassifier are the slices of the KIS info
// clients that kisAssetClassifier depends on; narrow interfaces so the routing
// (incl. the overseas exchange fallback) is unit-testable with fakes.
type domesticInfoClassifier interface {
	ClassifyAssetClass(ticker string) (string, error)
}

type overseasInfoClassifier interface {
	ClassifyAssetClass(excd, ticker string) (string, error)
}

// kisAssetClassifier routes a ticker to the domestic or overseas KIS info
// endpoint and reports whether it is an "etf" or a "stock". It satisfies
// services.AssetClassifier.
type kisAssetClassifier struct {
	domestic domesticInfoClassifier
	overseas overseasInfoClassifier
}

// _overseasEXCDFallback is the order in which US exchanges are tried when the
// stored exchange code does not pin a market (empty/unrecognized).
var _overseasEXCDFallback = []string{"NAS", "NYS", "AMS"}

func (k *kisAssetClassifier) ClassifyAssetClass(ticker, exchange string) (string, error) {
	if kis.IsDomesticTicker(ticker) {
		return k.domestic.ClassifyAssetClass(ticker)
	}
	if code, ok := overseasPriceEXCD(exchange); ok {
		return k.overseas.ClassifyAssetClass(code, ticker)
	}
	// Unknown/empty exchange: the stored code doesn't identify a market, so try
	// each US exchange in turn (up to 3× the API calls) and return the first that
	// resolves. KIS returns a business error (rt_cd != 0) for a wrong-market
	// lookup, so a non-nil error means "not on this exchange — try the next".
	var lastErr error
	for _, code := range _overseasEXCDFallback {
		ac, err := k.overseas.ClassifyAssetClass(code, ticker)
		if err == nil {
			return ac, nil
		}
		lastErr = err
	}
	return "", lastErr
}

// overseasPriceEXCD maps a stored long exchange code (NASD/NYSE/AMEX) to the
// short KIS info/price code (NAS/NYS/AMS). The bool is false when the code is
// empty or unrecognized, signalling the caller to fall back to trying each
// exchange in turn rather than silently assuming NASDAQ.
func overseasPriceEXCD(exchange string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(exchange)) {
	case "NASD", "NAS":
		return "NAS", true
	case "NYSE", "NYS":
		return "NYS", true
	case "AMEX", "AMS":
		return "AMS", true
	default:
		return "", false
	}
}

// buildAssetClassifier builds a KIS-backed asset classifier, or nil if KIS keys
// are absent (a nil classifier disables classification downstream).
func buildAssetClassifier(auth *kisAuth) services.AssetClassifier {
	if auth == nil {
		return nil
	}
	domesticInfoTrID := strings.TrimSpace(os.Getenv("KIS_DOMESTIC_INFO_TR_ID"))
	if domesticInfoTrID == "" {
		domesticInfoTrID = "CTPF1002R"
	}
	overseasInfoTrID := strings.TrimSpace(os.Getenv("KIS_OVERSEAS_INFO_TR_ID"))
	if overseasInfoTrID == "" {
		overseasInfoTrID = "CTPF1702R"
	}
	return &kisAssetClassifier{
		domestic: &kis.DomesticInfoClient{
			HTTP:      auth.httpClient,
			BaseURL:   auth.baseURL,
			AppKey:    auth.appKey,
			AppSecret: auth.appSecret,
			TrID:      domesticInfoTrID,
			CustType:  auth.custType,
			Manager:   auth.tokenManager,
		},
		overseas: &kis.OverseasInfoClient{
			HTTP:      auth.httpClient,
			BaseURL:   auth.baseURL,
			AppKey:    auth.appKey,
			AppSecret: auth.appSecret,
			TrID:      overseasInfoTrID,
			CustType:  auth.custType,
			Manager:   auth.tokenManager,
		},
	}
}

// buildBalanceClient returns a DomesticBalanceClient for the primary key, or nil if
// keys/account are absent. Requires auth.cano so the rebalance adapter has a fallback CANO.
func buildBalanceClient(auth *kisAuth) services.BalanceClient {
	if auth == nil || auth.cano == "" {
		return nil
	}
	log.Printf("KIS balance client initialized (env=%q)", auth.env) //nolint:gosec // env is operator-controlled, not user input
	return buildBalanceClientFromAuth(auth)
}

// buildBalanceClientFromAuth builds a DomesticBalanceClient for an extra key set.
// CANO is always passed per-call; no global account required.
func buildBalanceClientFromAuth(auth *kisAuth) services.BalanceClient {
	return &kis.DomesticBalanceClient{
		HTTP:      auth.httpClient,
		BaseURL:   auth.baseURL,
		AppKey:    auth.appKey,
		AppSecret: auth.appSecret,
		CustType:  auth.custType,
		Env:       auth.env,
		Manager:   auth.tokenManager,
	}
}

// buildKISAuthExtra builds a kisAuth for an extra API key set (id >= 2).
// Reads KIS_APP_KEY_{id} / KIS_APP_SECRET_{id}; inherits env/custType/baseURL/httpClient
// from the primary auth so all key sets share the same KIS environment.
func buildKISAuthExtra(id int, base *kisAuth) *kisAuth {
	suffix := fmt.Sprintf("_%d", id)
	appKey := strings.TrimSpace(os.Getenv("KIS_APP_KEY" + suffix))
	appSecret := strings.TrimSpace(os.Getenv("KIS_APP_SECRET" + suffix))
	if appKey == "" || appSecret == "" {
		return nil
	}
	authClient := &kis.AuthClient{
		HTTPClient: base.httpClient,
		BaseURL:    base.baseURL,
		AppKey:     appKey,
		AppSecret:  appSecret,
	}
	store := kis.NewFileTokenStore(fmt.Sprintf(".data/kis_token_%d.json", id))
	manager := kis.NewTokenManager(store, authClient, time.Minute)
	log.Printf("KIS balance client %d initialized (env=%q)", id, base.env) //nolint:gosec
	return &kisAuth{
		appKey:       appKey,
		appSecret:    appSecret,
		env:          base.env,
		custType:     base.custType,
		baseURL:      base.baseURL,
		httpClient:   base.httpClient,
		tokenManager: manager,
	}
}

// buildExchangeRate reads USD_KRW_RATE or EXIM_AUTH_KEY env vars.
// Priority: fixed rate > EXIM client > nil.
// When nil, USD holdings will show ₩0 for KRW values on the dashboard.
// Set USD_KRW_RATE=<rate> (e.g. 1380) or EXIM_AUTH_KEY to enable conversion.
func buildExchangeRate() *services.ExchangeRateService {
	if raw := strings.TrimSpace(os.Getenv("USD_KRW_RATE")); raw != "" {
		rate, err := numeric.FromString(raw)
		if err != nil {
			log.Printf("invalid USD_KRW_RATE: %v", err)
			return nil
		}
		log.Printf("exchange rate: fixed USD/KRW = %s", rate.String())
		return services.NewFixedExchangeRateService(rate)
	}

	if authKey := strings.TrimSpace(os.Getenv("EXIM_AUTH_KEY")); authKey != "" {
		eximClient := &services.EximClient{
			HTTPClient: &http.Client{Timeout: 10 * time.Second},
			BaseURL:    "https://oapi.koreaexim.go.kr",
			AuthKey:    authKey,
		}
		log.Printf("exchange rate: EXIM live feed enabled")
		return services.NewEximExchangeRateService(eximClient)
	}

	log.Printf("exchange rate: not configured (USD_KRW_RATE or EXIM_AUTH_KEY missing) — USD holdings will show ₩0 KRW value")
	return nil
}
