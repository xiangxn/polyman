package engine

import "github.com/xiangxn/go-polymarket-sdk/orders"

type Intent struct {
	StrategyID string
	MarketID   string
	TokenID    string
	Side       Side
	Price      float64
	Size       float64
	Type       orders.OrderType

	ReduceOnly bool
	TTL        int64
}

type Side uint8

const (
	BUY Side = iota
	SELL
)
