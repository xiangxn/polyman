package strategies

import (
	"context"
	"log"

	"github.com/xiangxn/polyman/internal/engine"
	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/model"

	"github.com/xiangxn/go-polymarket-sdk/orders"
)

type SimpleStrategy struct {
	bus *engine.EventBus
}

func (s *SimpleStrategy) SetEventBus(bus *engine.EventBus) {
	s.bus = bus
}

func (s *SimpleStrategy) OnTick(e marketdata.MockEvent) []model.Intent {
	if e.Price < 101 {
		return []model.Intent{
			{
				StrategyID: s.Name(),
				Market:     e.Market,
				Token:      e.Token,
				Side:       model.SideBuy,
				OrderType:  orders.GTC,
				Price:      e.Price,
				Size:       1,
			},
		}
	}
	return nil
}

func (s *SimpleStrategy) Name() string {
	return "SimpleStrategy"
}

func (s *SimpleStrategy) Run(ctx context.Context, executor engine.ExecutorController, dataCtrls map[string]engine.Controller) error {
	log.Println("SimpleStrategy running")
	priceCh, cancel := engine.SubscribeTyped[marketdata.MockEvent](
		s.bus,
		engine.EventPriceUpdate,
		1024,
	)

	for {
		select {
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		case tick := <-priceCh:
			intents := s.OnTick(tick)
			if len(intents) > 0 {
				executor.Submit(ctx, intents)
			}
		}
	}
}
