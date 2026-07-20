package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/kis"
)

// runKis dispatches "pm kis <verb>" subcommands (KIS-linked accounts only).
func runKis(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm kis order-cash -account NAME [-ticker T -price P]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "order-cash":
		return kisOrderCash(ctx, c, rest)
	default:
		return fmt.Errorf("unknown kis verb %q", verb)
	}
}

// buyableOutput is the JSON shape emitted by "pm kis order-cash".
type buyableOutput struct {
	Account             string `json:"account"`
	Ticker              string `json:"ticker,omitempty"`
	OrderableCash       string `json:"orderable_cash"`       // 주문가능현금
	OrderableSubstitute string `json:"orderable_substitute"` // 주문가능대용
	ReusableAmount      string `json:"reusable_amount"`      // 재사용가능금액
	NrcvbBuyAmount      string `json:"nrcvb_buy_amount"`     // 미수없는매수금액
	NrcvbBuyQty         string `json:"nrcvb_buy_qty"`        // 미수없는매수수량
	MaxBuyAmount        string `json:"max_buy_amount"`       // 최대매수금액
	MaxBuyQty           string `json:"max_buy_qty"`          // 최대매수수량
}

// kisOrderCash queries KIS 매수가능금액 (available buying power) for a
// KIS-linked account. -ticker/-price are optional; passing them fills in the
// max-buy quantity fields, otherwise only the cash figures are meaningful.
func kisOrderCash(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm kis order-cash", flag.ExitOnError)
	account := fs.String("account", "", "account name, exact or unique substring match (required)")
	ticker := fs.String("ticker", "", "domestic ticker for max-buy quantity (optional)")
	price := fs.String("price", "", "unit price for max-buy quantity; default market price (optional)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	pdno := strings.TrimSpace(*ticker)
	unpr := strings.TrimSpace(*price)
	// -price sizes a 지정가 max-buy quantity for a specific ticker; without a
	// ticker the PDNO is blank, so a 지정가 (ORD_DVSN 00) inquire-psbl-order is a
	// malformed request. Reject it up front, before touching KIS.
	if unpr != "" && pdno == "" {
		return fmt.Errorf("-price requires -ticker (a limit price only sizes a max-buy quantity for a specific ticker)")
	}

	acct, err := resolveAccountByName(ctx, c, *account)
	if err != nil {
		return err
	}
	if acct.KisAccountNo == nil || strings.TrimSpace(*acct.KisAccountNo) == "" {
		return fmt.Errorf("account %q has no KIS account number linked", acct.Name)
	}
	cano, acntPrdtCd, err := normalizeKisAccountNo(*acct.KisAccountNo)
	if err != nil {
		return err
	}

	client, err := c.BuildKISBuyableClient(acct.KisAPIKeyID)
	if err != nil {
		return fmt.Errorf("build kis buyable client: %w", err)
	}

	// A price is only honoured by KIS under 지정가 (ORD_DVSN 00); under the
	// default 시장가 (01) the ORD_UNPR is ignored, so max-buy-qty would reflect
	// a market order regardless. Query as limit whenever a price is supplied so
	// -price actually sizes a limit order.
	ordDvsn := ""
	if unpr != "" {
		ordDvsn = "00"
	}
	result, err := client.FetchBuyable(cano, acntPrdtCd, pdno, unpr, ordDvsn)
	if err != nil {
		return fmt.Errorf("kis order-cash: %w", err)
	}

	return printJSON(toBuyableOutput(acct.Name, pdno, result))
}

func toBuyableOutput(account, ticker string, b kis.BuyableAmount) buyableOutput {
	return buyableOutput{
		Account:             account,
		Ticker:              ticker,
		OrderableCash:       b.OrderableCash.String(),
		OrderableSubstitute: b.OrderableSubstitute.String(),
		ReusableAmount:      b.ReusableAmount.String(),
		NrcvbBuyAmount:      b.NrcvbBuyAmount.String(),
		NrcvbBuyQty:         b.NrcvbBuyQty.String(),
		MaxBuyAmount:        b.MaxBuyAmount.String(),
		MaxBuyQty:           b.MaxBuyQty.String(),
	}
}
