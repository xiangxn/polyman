package engine

import (
	"context"
	"fmt"

	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/model"
	"github.com/xiangxn/polyman/internal/order"
	"github.com/xiangxn/polyman/internal/position"
	"github.com/xiangxn/polyman/internal/strategies"
)

type Engine struct {
	md       marketdata.MarketData
	strategy strategies.Strategy
	orderer  order.Executor
	pos      *position.Manager

	intentCh chan model.Intent
}

func New(
	md marketdata.MarketData,
	strategy strategies.Strategy,
	orderer order.Executor,
	pos *position.Manager,
) *Engine {
	return &Engine{md, strategy, orderer, pos, make(chan model.Intent, 1024)}
}

func (e *Engine) Run(ctx context.Context) error {
	// 1️⃣ 初始化策略
	if initer, ok := e.strategy.(strategies.InitnableStrategy); ok {
		ctrl, ok := e.md.(marketdata.MarketDataController)
		if !ok {
			if err := initer.Init(ctx, nil); err != nil {
				return err
			}
		} else {
			if err := initer.Init(ctx, ctrl); err != nil {
				return err
			}
		}
	}

	// 2️⃣ 如果策略实现了 Run，启动其生命周期 goroutine
	if runner, ok := e.strategy.(strategies.RunnableStrategy); ok {
		go func() {
			if err := runner.Run(ctx); err != nil && err != context.Canceled {
				// 这里非常关键：
				// 策略 Run 出错，应该让整个系统停下来
				panic(fmt.Errorf("strategy run failed: %w", err))
			}
		}()
	}

	// 3️⃣ 行情 → 策略 → 下单 主循环
	ticks := e.md.Subscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case tick, ok := <-ticks:
			if !ok {
				return fmt.Errorf("market data closed")
			}

			intents := e.strategy.OnTick(tick)
			for _, in := range intents {
				if !e.pos.CanOpen(in) {
					continue
				}
				if err := e.orderer.Submit(ctx, in); err != nil {
					// executor 满 / ctx cancel
					// 这里可以打 metrics
				}
			}
		}
	}
}
