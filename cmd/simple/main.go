package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xiangxn/polyman/internal/common"
	"github.com/xiangxn/polyman/internal/engine"
	"github.com/xiangxn/polyman/internal/executor"
	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/position"
	"github.com/xiangxn/polyman/internal/strategies"
)

func main() {
	log.Println("🚀 polyman system starting...")
	ctx, cancel := context.WithCancel(context.Background())

	md := marketdata.NewMockMarketData()
	strat := &strategies.SimpleStrategy{}
	orderer := executor.NewSimpleExecutor()
	pos := position.NewManager(1000)

	eng := engine.New(md, strat, orderer, pos)

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
