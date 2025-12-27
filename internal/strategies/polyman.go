package strategies

import (
	"context"
	"polyman/internal/model"

	"github.com/xiangxn/go-polymarket-sdk/orders"
)

type PolymanStrategy struct{}

func (s *PolymanStrategy) OnTick(t model.Tick) []model.Intent {
	if t.Price < 101 {
		return []model.Intent{
			{
				Market:    t.Market,
				Token:     t.Token,
				Side:      orders.BUY,
				OrderType: orders.GTC,
				Price:     t.Price,
				Size:      1,
			},
		}
	}
	return nil
}

func (s *PolymanStrategy) Init(ctx context.Context) error {
	return nil
}

// 可选的周期性处理，或市场结束处理
func (s *PolymanStrategy) Run(ctx context.Context) error {
	return nil
}
