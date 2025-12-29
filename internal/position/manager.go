package position

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/xiangxn/polyman/internal/executor"
	"github.com/xiangxn/polyman/internal/model"
)

type PositionManager struct {
	mu sync.RWMutex

	// key = marketID:tokenID
	positions map[string]*model.Position

	// Fill 幂等
	appliedFills map[string]struct{}

	// 冻结仓位：OrderID -> FrozenPosition
	frozen map[string]model.FrozenPosition
}

func NewManager() *PositionManager {
	return &PositionManager{
		positions:    make(map[string]*model.Position),
		appliedFills: make(map[string]struct{}),
		frozen:       make(map[string]model.FrozenPosition),
	}
}

/* =========================
   Query / Risk
   ========================= */

func (m *PositionManager) CanOpen(in model.Intent) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	maxSize := float64(1000)

	current := m.getEffectiveSizeLocked(in.Market, in.Token, in.Side)
	return current+in.Size <= maxSize
}

func (m *PositionManager) GetPosition(marketID, tokenID string) model.PositionView {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getPositionViewLocked(marketID, tokenID, 0, false)
}

func (m *PositionManager) GetPositionWithPrice(
	marketID,
	tokenID string,
	marketPrice float64,
) model.PositionView {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getPositionViewLocked(marketID, tokenID, marketPrice, true)
}

/* =========================
   Event Entry (ONLY public writer)
   ========================= */

func (pm *PositionManager) OnEvent(evt executor.ExecutionEvent) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	switch evt.Type {

	case executor.EventFill:
		if evt.Fill != nil {
			pm.onFillLocked(*evt.Fill)
		}

	case executor.EventReject:
		pm.unfreezeLocked(evt.Intent.OrderID, evt.Err)

	case executor.EventCancel:
		pm.unfreezeLocked(evt.Intent.OrderID, nil)

	case executor.EventExpire:
		pm.unfreezeLocked(evt.Intent.OrderID, errors.New("order expired"))
	}
}

/* =========================
   Internal Mutations (LOCKED)
   ========================= */

func (m *PositionManager) onFillLocked(fill model.Fill) {
	if fill.FillID == "" {
		log.Printf("[Position] empty FillID, skip: %+v", fill)
		return
	}

	// ---------- 1. 幂等 ----------
	if _, seen := m.appliedFills[fill.FillID]; seen {
		return
	}
	m.appliedFills[fill.FillID] = struct{}{}

	// ---------- 2. Apply Position ----------
	key := fill.MarketID + ":" + fill.TokenID
	pos, ok := m.positions[key]
	if !ok {
		pos = &model.Position{
			MarketID: fill.MarketID,
			TokenID:  fill.TokenID,
		}
		m.positions[key] = pos
	}

	if err := pos.ApplyFill(fill); err != nil {
		log.Printf("[Position] apply fill failed: %v, fill=%+v", err, fill)
	}

	// ---------- 3. 解冻 ----------
	if frozen, ok := m.frozen[fill.OrderID]; ok {
		frozen.Size -= fill.Size
		if frozen.Size <= 0 {
			delete(m.frozen, fill.OrderID)
		} else {
			m.frozen[fill.OrderID] = frozen
		}
	}
}

func (m *PositionManager) unfreezeLocked(orderID string, err error) {
	if orderID == "" {
		return
	}

	if err != nil {
		log.Printf("[PositionManager] order end: %s, err=%v", orderID, err)
	}

	delete(m.frozen, orderID)
}

/* =========================
   Freeze (explicit call)
   ========================= */

func (m *PositionManager) Freeze(intent model.Intent) error {
	if intent.OrderID == "" {
		return fmt.Errorf("empty OrderID")
	}
	if intent.Size <= 0 {
		return fmt.Errorf("invalid size")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 幂等
	if _, exists := m.frozen[intent.OrderID]; exists {
		return nil
	}

	m.frozen[intent.OrderID] = model.FrozenPosition{
		OrderID:   intent.OrderID,
		MarketID:  intent.Market,
		TokenID:   intent.Token,
		Side:      intent.Side,
		Size:      intent.Size,
		CreatedAt: time.Now().UnixMilli(),
	}

	return nil
}

/* =========================
   Internal Helpers (LOCKED)
   ========================= */

func (m *PositionManager) getEffectiveSizeLocked(
	marketID,
	tokenID string,
	side model.Side,
) float64 {
	var size float64

	if pos, ok := m.positions[marketID+":"+tokenID]; ok {
		if pos.Side == side {
			size += pos.Size
		}
	}

	for _, f := range m.frozen {
		if f.MarketID == marketID &&
			f.TokenID == tokenID &&
			f.Side == side {
			size += f.Size
		}
	}

	return size
}

func (m *PositionManager) getPositionViewLocked(
	marketID,
	tokenID string,
	marketPrice float64,
	withPrice bool,
) model.PositionView {

	if pos, ok := m.positions[marketID+":"+tokenID]; ok {
		view := model.PositionView{
			Size:     pos.Size,
			Side:     pos.Side,
			AvgPrice: pos.AvgPrice,
		}

		if withPrice {
			view.PNL = pos.CalcUnrealized(marketPrice)
		}

		return view
	}

	return model.PositionView{
		Size:     0,
		Side:     model.SideUnknown,
		AvgPrice: 0,
		PNL:      0,
	}
}
