package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xiangxn/polyman/internal/common"
	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/engine"
	"github.com/xiangxn/polyman/internal/executor"
	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/position"
	"github.com/xiangxn/polyman/internal/strategies"
	"github.com/xiangxn/polyman/internal/version"

	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf(
			"polyman %s (%s) %s\n",
			version.Version,
			version.Commit,
			version.Date,
		)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// log.Printf("config: %+v", cfg)

	log.Printf("🚀 [%s] polyman system starting...", version.Version)
	pmClient := pm.NewClient(cfg.OwnerKey, &cfg.PmSDK)
	ctx, cancel := context.WithCancel(context.Background())

	md := marketdata.NewPolymarketData(cfg.PmSDK.Polymarket.ClobWSBaseURL, pmClient)
	strategy := &strategies.PolymanStrategy{MarketSlug: cfg.MarketSlug}
	executor := executor.NewLiveExecutor(pmClient, nil, cfg.OrderEngine, cfg.PmSDK.Polymarket, cfg.Balance)
	pos := position.NewManager(100)

	eng := engine.New(md, strategy, executor, pos)

	g := common.NewSafeGroup(ctx)
	g.Go("market-data", md.Run)
	g.Go("executor", executor.Run)
	g.Go("engine", eng.Run)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Println("🛑 received signal:", sig)
		cancel()
	case <-ctx.Done():
	}

	if err := g.Wait(); err != nil {
		log.Println("polyman stopped:", err)
	}
	log.Println("✅ polyman system stopped")
}
