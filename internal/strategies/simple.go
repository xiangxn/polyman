package strategies

import (
	"github.com/xiangxn/polyman/internal/model"

	"github.com/xiangxn/go-polymarket-sdk/orders"
)

type SimpleStrategy struct{}

func (s *SimpleStrategy) OnTick(t model.Tick) []model.Intent {
	if t.Price < 101 {
		return []model.Intent{
			{
				Market:    t.Market,
				Token:     t.Token,
				Side:      model.SideLong,
				OrderType: orders.GTC,
				Price:     t.Price,
				Size:      1,
			},
		}
	}
	return nil
}
