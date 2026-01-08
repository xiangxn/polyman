package model

type MarketEventType int

const (
	EventExternalPrice EventType = iota
	EventInternalPrice
	EventOrderBook
)

type MarketEvent struct {
	Source string
	Symbol string
	Type   MarketEventType
	Data   any
	Ts     int64
}
