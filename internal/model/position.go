package model

import (
	"fmt"
	"math"
	"time"
)

type Side int

const (
	SideUnknown Side = iota
	SideLong
	SideShort
)

func (s Side) Opposite() Side {
	switch s {
	case SideLong:
		return SideShort
	case SideShort:
		return SideLong
	default:
		return SideUnknown
	}
}

type Position struct {
	MarketID string
	TokenID  string

	Side Side
	Size float64

	AvgPrice float64

	RealizedPNL   float64
	UnrealizedPNL float64

	OpenedAt  int64
	UpdatedAt int64
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

func (p *Position) ApplyFill(fill Fill) error {
	if fill.Size <= 0 {
		return fmt.Errorf("invalid fill size: %d", fill.Size)
	}
	if fill.Price <= 0 {
		return fmt.Errorf("invalid fill price: %f", fill.Price)
	}
	if fill.Side != SideLong && fill.Side != SideShort {
		return fmt.Errorf("invalid fill side")
	}

	now := fill.Time
	if now == 0 {
		now = time.Now().UnixMilli()
	}

	// ---------- 1. 当前无仓位：直接开仓 ----------
	if p.Size == 0 {
		p.Side = fill.Side
		p.Size = fill.Size
		p.AvgPrice = fill.Price
		p.OpenedAt = now
		p.UpdatedAt = now
		return nil
	}

	// ---------- 2. 同方向：加仓 ----------
	if p.Side == fill.Side {
		newSize := p.Size + fill.Size
		p.AvgPrice = (p.AvgPrice*p.Size + fill.Price*fill.Size) / newSize
		p.Size = newSize
		p.UpdatedAt = now
		return nil
	}

	// ---------- 3. 反方向：平仓 / 反手 ----------
	// 计算可平掉的数量
	closeSize := math.Min(p.Size, fill.Size)

	// 计算 Realized PnL
	var pnl float64
	if p.Side == SideLong {
		pnl = float64(closeSize) * (fill.Price - p.AvgPrice)
	} else {
		pnl = float64(closeSize) * (p.AvgPrice - fill.Price)
	}
	p.RealizedPNL += pnl

	remaining := fill.Size - closeSize

	if remaining == 0 {
		// 3.1 刚好平完
		p.Size -= closeSize
		if p.Size == 0 {
			p.Side = SideUnknown
			p.AvgPrice = 0
			p.OpenedAt = 0
		}
		p.UpdatedAt = now
		return nil
	}

	// 3.2 反手：先平完，再用剩余的开新仓
	p.Side = fill.Side
	p.Size = remaining
	p.AvgPrice = fill.Price
	p.OpenedAt = now
	p.UpdatedAt = now

	return nil
}

func (p *Position) CalcUnrealized(marketPrice float64) float64 {
	if p.Size == 0 {
		return 0
	}
	return float64(p.Size) * (marketPrice - p.AvgPrice)
}

type FrozenPosition struct {
	OrderID  string
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
