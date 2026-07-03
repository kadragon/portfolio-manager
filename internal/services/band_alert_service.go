package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

const bandAlertInterval = 24 * time.Hour

// bandDiagSource yields the current band diagnostics (summary → CheckBands).
type bandDiagSource interface {
	diagnostics(ctx context.Context) ([]models.GroupDiagnostic, error)
}

// BandAlertService periodically checks rebalance-band state and posts a webhook
// notification when the breach set changes. Monitoring is what makes
// threshold-based rebalancing work ("look often, trade rarely") — the user no
// longer needs to remember to open the rebalance page.
type BandAlertService struct {
	source     bandDiagSource
	webhookURL string
	client     *http.Client
	// lastSig is the signature of the breach set already alerted on. A breach
	// alerts once and stays silent while it persists; clearing resets the
	// state so a later breach alerts again. In-memory only — a restart may
	// re-alert once, which is acceptable.
	lastSig string
}

// NewBandAlertService wires the production diagnostic source. webhookURL must
// be non-empty (the container skips construction otherwise).
func NewBandAlertService(
	portfolio *PortfolioService,
	groups *repositories.GroupRepository,
	rebalance *RebalanceService,
	webhookURL string,
) *BandAlertService {
	return &BandAlertService{
		source:     &portfolioBandSource{portfolio: portfolio, groups: groups, rebalance: rebalance},
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Start checks immediately, then repeats daily until ctx is cancelled.
func (s *BandAlertService) Start(ctx context.Context) {
	if err := s.CheckOnce(ctx); err != nil {
		log.Printf("band alert: %v", err)
	}
	t := time.NewTicker(bandAlertInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := s.CheckOnce(ctx); err != nil {
				log.Printf("band alert: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// CheckOnce fetches diagnostics and notifies if the breach set changed.
func (s *BandAlertService) CheckOnce(ctx context.Context) error {
	diags, err := s.source.diagnostics(ctx)
	if err != nil {
		return fmt.Errorf("diagnostics: %w", err)
	}
	sig := breachSignature(diags)
	if sig == s.lastSig {
		return nil
	}
	if sig == "" {
		s.lastSig = ""
		return nil
	}
	if err := s.notify(ctx, breachMessage(diags)); err != nil {
		return fmt.Errorf("notify: %w", err)
	}
	s.lastSig = sig
	return nil
}

func (s *BandAlertService) notify(ctx context.Context, msg string) error {
	// "text" is read by Slack-style webhooks, "content" by Discord-style ones;
	// each platform ignores the key it does not know.
	body, err := json.Marshal(map[string]string{"text": msg, "content": msg})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		// url.Error echoes the request URL, and webhook URLs carry secret
		// tokens — strip it so the logged error stays secret-free.
		var ue *url.Error
		if errors.As(err, &ue) {
			return fmt.Errorf("webhook request: %w", ue.Err)
		}
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}

// breachSignature encodes the breach set (group + direction, diagnostic order)
// so alerts fire exactly when the set changes. Empty when nothing is breached.
func breachSignature(diags []models.GroupDiagnostic) string {
	var parts []string
	for _, d := range diags {
		switch {
		case d.IsUpperBreached:
			parts = append(parts, d.RebalanceGroupName+":상단")
		case d.IsLowerBreached:
			parts = append(parts, d.RebalanceGroupName+":하단")
		}
	}
	return strings.Join(parts, ",")
}

func breachMessage(diags []models.GroupDiagnostic) string {
	var b strings.Builder
	b.WriteString("⚠️ 리밸런싱 밴드 이탈 감지")
	for _, d := range diags {
		if !d.IsUpperBreached && !d.IsLowerBreached {
			continue
		}
		dir := "상단"
		if d.IsLowerBreached {
			dir = "하단"
		}
		fmt.Fprintf(&b, "\n- %s: 현재 %.2f%% / 목표 %.2f%% (밴드 ±%.2f%%) — %s 이탈",
			d.RebalanceGroupName, floatOf(d.CurrentPct.Decimal), floatOf(d.TargetPct.Decimal), floatOf(d.BandPct.Decimal), dir)
	}
	b.WriteString("\n/rebalance 에서 계획을 확인하세요.")
	return b.String()
}

// portfolioBandSource assembles CheckBands inputs from live services.
type portfolioBandSource struct {
	portfolio *PortfolioService
	groups    *repositories.GroupRepository
	rebalance *RebalanceService
}

func (p *portfolioBandSource) diagnostics(ctx context.Context) ([]models.GroupDiagnostic, error) {
	summary, err := p.portfolio.GetPortfolioSummary(ctx, false)
	if err != nil {
		return nil, err
	}
	groups, err := p.groups.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return p.rebalance.CheckBands(*summary, groups)
}
