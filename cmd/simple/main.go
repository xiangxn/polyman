package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xiangxn/polyman/internal/engine"
	"github.com/xiangxn/polyman/internal/executor"
	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/strategies"
)

func main() {
	log.Println("🚀 polyman system starting...")
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	md := marketdata.NewMockMarketData()
	strat := &strategies.SimpleStrategy{}
	orderer := executor.NewSimpleExecutor()

	eng := engine.NewEngine([]engine.Feed{md}, []engine.Strategy[engine.Controller]{strat}, orderer)

	go func() {
		sig := <-sigCh
		log.Println("🛑 received signal:", sig)
		cancel()
	}()

	if err := eng.Run(ctx); err != nil {
		log.Println("⚠️ Engine stopped with error:", err)
	}

	log.Println("✅ polyman system stopped")
}
