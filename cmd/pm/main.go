// Command pm is the CLI entrypoint that replaces the removed web UI. Claude
// Code skills (portfolio-data, portfolio-sync) invoke it one subcommand at a
// time; each invocation boots its own container and exits. Output is
// indented JSON on stdout, matching cmd/rebalance-order's convention.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kadragon/portfolio-manager/internal/container"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	resource := os.Args[1]
	args := os.Args[2:]

	if resource == "-h" || resource == "--help" || resource == "help" {
		usage()
		return
	}

	c, err := container.New("")
	if err != nil {
		fatal("init container: %v", err)
	}

	ctx := context.Background()

	var runErr error
	switch resource {
	case "account":
		runErr = runAccount(ctx, c, args)
	case "group":
		runErr = runGroup(ctx, c, args)
	case "stock":
		runErr = runStock(ctx, c, args)
	case "deposit":
		runErr = runDeposit(ctx, c, args)
	case "holding":
		runErr = runHolding(ctx, c, args)
	case "sync":
		runErr = runSync(ctx, c, args)
	case "classify-stocks":
		runErr = runClassifyStocks(ctx, c, args)
	case "dashboard":
		runErr = runDashboard(ctx, c, args)
	case "price":
		runErr = runPrice(ctx, c, args)
	case "price-sync":
		runErr = runPriceSync(ctx, c, args)
	case "price-backfill":
		runErr = runPriceBackfill(ctx, c, args)
	default:
		fmt.Fprintf(os.Stderr, "unknown resource %q\n\n", resource)
		usage()
		_ = c.Close()
		os.Exit(2)
	}

	if runErr != nil {
		_ = c.Close()
		fatal("%v", runErr)
	}
	_ = c.Close()
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: pm <resource> <verb> [flags]

resources:
  account          list | add -name -cash | update -id [-name -cash -kis-account-no -kis-api-key-id -account-type -toss-account-seq] | delete -id | set-cash -id -cash
  group            list | add -name -target | update -id [-name -target] | delete -id
  stock            list [-group] | add -group -ticker | update -id [-ticker -exchange -name -asset-class -security-group] | move -id -group | delete -id
  deposit          list | add -amount -date [-note] | update -id [-amount -date -note] | delete -id
  holding          list -account | add -account -stock -qty | add-by-ticker -account -ticker -qty | bulk -account -updates | update -id -qty | delete -id
  sync             -account NAME [-confirm-empty]   (KIS/Toss account sync, routed by account link)
  classify-stocks  backfill asset_class via KIS
  dashboard        [-no-change-rates] [-sort {value,1d,1m,6m,1y}] [-asc]   (portfolio summary)
  price            list -ticker T [-from YYYY-MM-DD] [-to YYYY-MM-DD] [-limit N]   (read cached daily closes)
  price-sync       refresh stock prices once
  price-backfill   -ticker T -from YYYY-MM-DD -to YYYY-MM-DD   (backfill historical closes for one ticker)

Every subcommand prints indented JSON to stdout and exits non-zero on error.`)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
