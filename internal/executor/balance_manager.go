package executor

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/utils"
)

type BalanceManager struct {
	mu      sync.RWMutex
	balance utils.ERC20Info
	client  *ethclient.Client
	config  *config.BalanceConfig
	frozen  float64 // 冻结的金额
}

func NewBalanceManager(config *config.BalanceConfig) *BalanceManager {
	client, err := ethclient.Dial(config.ChainRPC)
	if err != nil {
		panic(err)
	}
	return &BalanceManager{
		client:  client,
		config:  config,
		balance: utils.ERC20Info{Balance: new(big.Int).SetInt64(0)},
	}
}

func (m *BalanceManager) Run(ctx context.Context) error {
	log.Println("[BalanceManager] Run start")
	timer := time.NewTicker(time.Duration(m.config.Interval) * time.Second)
	defer timer.Stop()

	m.syncOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Println("[BalanceManager] Run exit")
			return ctx.Err()
		case <-timer.C:
			m.syncOnce(ctx)
		}
	}
}

func (m *BalanceManager) syncOnce(ctx context.Context) {
	info, err := utils.FetchERC20InfoMulticall3(ctx, m.client, m.config.ChainID, common.HexToAddress(m.config.TokenAddress), common.HexToAddress(*m.config.FunderAddress))
	if err != nil {
		log.Println("[BalanceManager] FetchERC20InfoMulticall3 error:", err)
		return
	}
	log.Printf("[BalanceManager] %s: %s", info.Token.String(), info.String())
	m.mu.Lock()
	m.balance = *info
	m.mu.Unlock()
}

func (m *BalanceManager) GetBalance() *utils.ERC20Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &m.balance
}

func (m *BalanceManager) AvailableBalance() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	min := m.balance.Float() - m.frozen - m.config.MinBalance
	if min < 0 {
		return 0
	}
	return min
}

func (m *BalanceManager) Freeze(amount float64) error {
	if amount < 0 {
		return fmt.Errorf("invalid amount")
	}

	if amount > m.AvailableBalance() {
		return fmt.Errorf("insufficient available balance")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.frozen += amount
	return nil
}

func (m *BalanceManager) Unfreeze(amount float64) {
	if amount < 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.frozen -= amount
}
