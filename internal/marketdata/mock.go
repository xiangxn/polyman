package marketdata

import (
	"context"
	"polyman/internal/model"
	"time"
)

type MockMarketData struct {
	ch chan model.Tick
}

func NewMockMarketData() *MockMarketData {
	return &MockMarketData{
		ch: make(chan model.Tick, 100),
	}
}

func (m *MockMarketData) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.ch <- model.Tick{
				Market:    "BTC",
				Token:     "0",
				Price:     100,
				Timestamp: time.Now().UnixMilli(),
			}
		case <-ctx.Done():
			close(m.ch)
			return ctx.Err()
		}
	}
}

func (m *MockMarketData) Subscribe() <-chan model.Tick {
	return m.ch
}
