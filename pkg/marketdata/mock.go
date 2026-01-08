package marketdata

import (
	"context"
	"time"

	"github.com/xiangxn/polyman/pkg/engine"
)

type MockMarketData struct {
	bus *engine.EventBus
}

type MockEvent struct {
	Market string
	Token  string
	Price  float64
	Ts     int64
}

func (m MockEvent) Type() engine.EventType {
	return engine.EventPriceUpdate
}

func (m MockEvent) Time() int64 {
	return m.Ts
}

func NewMockMarketData() *MockMarketData {
	return &MockMarketData{}
}

func (m *MockMarketData) SetEventBus(bus *engine.EventBus) {
	m.bus = bus
}

func (m *MockMarketData) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.bus.Publish(MockEvent{
				Market: "BTC",
				Token:  "0",
				Price:  100,
				Ts:     time.Now().UnixMilli(),
			})
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
