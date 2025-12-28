package position

import (
	"sync"

	"github.com/xiangxn/polyman/internal/model"
)

type Manager struct {
	mu       sync.Mutex
	position map[string]float64
}

func NewManager() *Manager {
	return &Manager{
		position: make(map[string]float64),
	}
}

func (m *Manager) CanOpen(in model.Intent) bool {
	return true // 简化：永远允许
}
