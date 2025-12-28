package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xiangxn/polyman/internal/common"
	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/engine"
	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/order"
	"github.com/xiangxn/polyman/internal/position"
	"github.com/xiangxn/polyman/internal/strategies"

	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Println("🚀 polyman system starting...")
	pmClient := pm.NewClient(cfg.OwnerKey, &cfg.PmSDK)
	ctx, cancel := context.WithCancel(context.Background())

	md := marketdata.NewPolymarketData(cfg.PmSDK.Polymarket.ClobWSBaseURL, pmClient)
	strategy := &strategies.PolymanStrategy{MarketSlug: cfg.MarketSlug}
	orderer := order.NewConcurrentExecutor(pmClient, cfg.OrderEngine)
	pos := position.NewManager()

	eng := engine.New(md, strategy, orderer, pos)

	g := common.NewSafeGroup(ctx)
	g.Go("market-data", md.Run)
	g.Go("orderer", orderer.Run)
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
