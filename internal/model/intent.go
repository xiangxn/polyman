package model

import "github.com/xiangxn/go-polymarket-sdk/orders"

type Intent struct {
	Market    string
	Token     string
	Side      Side
	OrderType orders.OrderType
	Price     float64
	Size      float64
}
