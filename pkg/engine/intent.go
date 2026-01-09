package engine

import (
	"github.com/polymarket/go-order-utils/pkg/model"
	"github.com/xiangxn/go-polymarket-sdk/orders"
)

type Intent struct {
	StrategyID string
	MarketID   string
	TokenID    string
	Side       model.Side
	Price      float64
	Size       float64
	Type       orders.OrderType

	ReduceOnly bool
	TTL        int64
}
