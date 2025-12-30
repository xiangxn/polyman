package position

import (
	"fmt"
	"math"
	"time"

	"github.com/xiangxn/polyman/internal/model"
)

type Position struct {
	MarketID string
	TokenID  string

	Side model.Side
	Size float64

	AvgPrice float64
	TotalFee float64

	RealizedPNL float64

	OpenedAt  int64
	UpdatedAt int64
}

func (p *Position) ApplyFill(fill model.Fill) error {
	if fill.Size <= 0 {
		return fmt.Errorf("invalid fill size: %f", fill.Size)
	}
	if fill.Price <= 0 {
		return fmt.Errorf("invalid fill price: %f", fill.Price)
	}
	if fill.Side != model.SideBuy && fill.Side != model.SideSell {
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
		p.AvgPrice = (fill.Price*fill.Size + fill.Fee) / fill.Size
		p.OpenedAt = now
		p.UpdatedAt = now
		p.TotalFee = fill.Fee
		return nil
	}

	// ---------- 2. 同方向：加仓 ----------
	if p.Side == fill.Side {
		newSize := p.Size + fill.Size
		p.AvgPrice = (p.AvgPrice*p.Size + fill.Price*fill.Size + fill.Fee) / newSize
		p.Size = newSize
		p.UpdatedAt = now
		p.TotalFee += fill.Fee
		return nil
	}

	// ---------- 3. 反方向：平仓 / 反手 ----------
	// 计算可平掉的数量
	closeSize := math.Min(p.Size, fill.Size)

	// 计算 Realized PnL
	var pnl float64
	if p.Side == model.SideBuy {
		pnl = float64(closeSize) * (fill.Price - p.AvgPrice)
	} else {
		pnl = float64(closeSize) * (p.AvgPrice - fill.Price)
	}
	p.RealizedPNL += pnl - fill.Fee
	p.TotalFee += fill.Fee

	p.Size -= closeSize

	remaining := fill.Size - closeSize
	// 3.1 刚好平完
	if remaining == 0 {
		if p.Size == 0 {
			p.Side = model.SideUnknown
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
	if p.Side == model.SideBuy {
		return p.Size * (marketPrice - p.AvgPrice)
	}
	return p.Size * (p.AvgPrice - marketPrice)
}
