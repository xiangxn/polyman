package engine

type EventType uint8

const (
	EventMarketSnapshot EventType = iota
	EventOrderbookUpdate
	EventTrade
	EventMarketResolved

	EventPriceUpdate
	EventGameUpdate
	EventTimeProgress

	EventPositionUpdate
	EventRiskAlert
)

type Event interface {
	Type() EventType
	Time() int64
}
