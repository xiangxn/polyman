package model

import (
	"github.com/xiangxn/go-polymarket-sdk/orders"
)

type Intent struct {
	StrategyID string
	Market     string
	Token      string
	Side       Side
	Price      float64
	Size       float64
	OrderType  orders.OrderType

	// 扩展字段, Intent的有效时间,如果执行时已经超过TTL则不执行
	TTL int64
}
