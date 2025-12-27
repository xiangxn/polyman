package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"polyman/internal/common"
	"polyman/internal/config"
	"polyman/internal/engine"
	"polyman/internal/marketdata"
	"polyman/internal/order"
	"polyman/internal/position"
	"polyman/internal/strategies"
	"syscall"

	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
)

func main() {
	log.Println("🚀 polyman system starting...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	pmClient := pm.NewClient(cfg.OwnerKey, &cfg.PmSDK)
	ctx, cancel := context.WithCancel(context.Background())

	md := marketdata.NewPriceManager(cfg.PmSDK.Polymarket.ClobWSBaseURL)
	strategy := &strategies.PolymanStrategy{}
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
