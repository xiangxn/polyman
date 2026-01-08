package position

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/xiangxn/polyman/internal/model"
)

type PositionManager struct {
	mu sync.RWMutex

	// key = marketID:tokenID
	positions map[string]*Position

	// Fill 幂等
	appliedFills map[string]struct{}

	// 冻结仓位：OrderID -> FrozenPosition
	frozen map[string]model.FrozenPosition

	maxSize float64
}

func NewManager(maxSize float64) *PositionManager {
	return &PositionManager{
		positions:    make(map[string]*Position),
		appliedFills: make(map[string]struct{}),
		frozen:       make(map[string]model.FrozenPosition),
		maxSize:      maxSize,
	}
}

/*
检查仓位限制
*/
func (m *PositionManager) CanOpen(in model.Intent) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	current := m.getEffectiveSizeLocked(in.Market, in.Token, in.Side)
	return current+in.Size <= m.maxSize
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

func (pm *PositionManager) OnEvent(evt model.ExecutionEvent) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	switch evt.Type {

	case model.EventFill:
		if evt.Fill != nil {
			pm.onFillLocked(*evt.Fill)
		}

	case model.EventReject:
		pm.unfreezeLocked(evt.Intent, evt.Err)

	case model.EventCancel:
		pm.unfreezeLocked(evt.Intent, nil)

	case model.EventExpire:
		pm.unfreezeLocked(evt.Intent, errors.New("order expired"))
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
		pos = &Position{
			MarketID: fill.MarketID,
			TokenID:  fill.TokenID,
		}
		m.positions[key] = pos
	}

	if err := pos.ApplyFill(fill); err != nil {
		log.Printf("[Position] apply fill failed: %v, fill=%+v", err, fill)
	}

	// ---------- 3. 解冻 ----------
	if frozen, ok := m.frozen[key]; ok {
		frozen.Size -= fill.Size
		if frozen.Size <= 0 {
			delete(m.frozen, key)
		} else {
			m.frozen[key] = frozen
		}
	}
}

func (m *PositionManager) unfreezeLocked(intent *model.Intent, err error) {
	if intent == nil {
		return
	}

	key := intent.Market + ":" + intent.Token

	if err != nil {
		log.Printf("[PositionManager] key end: %s, err=%v", key, err)
	}

	delete(m.frozen, key)
}

/*
Freeze (explicit call)
*/
func (m *PositionManager) Freeze(intent model.Intent) error {
	if intent.Size <= 0 {
		return fmt.Errorf("invalid size")
	}
	if intent.Side != model.SideBuy {
		// only buy side can be frozen
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := intent.Market + ":" + intent.Token
	// 幂等
	if frozen, exists := m.frozen[key]; exists {
		frozen.Size += intent.Size
		m.frozen[key] = frozen
	} else {
		m.frozen[key] = model.FrozenPosition{
			MarketID:  intent.Market,
			TokenID:   intent.Token,
			Side:      intent.Side,
			Size:      intent.Size,
			CreatedAt: time.Now().UnixMilli(),
		}
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
	size := 0.0

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
