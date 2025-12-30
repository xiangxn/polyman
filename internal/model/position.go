package model

type Side int

const (
	SideUnknown Side = iota
	SideBuy
	SideSell
)

func (s Side) Opposite() Side {
	switch s {
	case SideBuy:
		return SideSell
	case SideSell:
		return SideBuy
	default:
		return SideUnknown
	}
}

type Fill struct {
	FillID   string // 平台返回的trade id
	OrderID  string
	MarketID string
	TokenID  string

	Side  Side
	Price float64
	Size  float64

	Fee  float64
	Time int64
}

type PositionView struct {
	Size     float64
	Side     Side
	AvgPrice float64
	PNL      float64
}

type FrozenPosition struct {
	MarketID string
	TokenID  string

	Side Side
	Size float64

	CreatedAt int64
}

type EffectivePosition struct {
	Side Side
	Size float64
}
