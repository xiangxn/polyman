package executor

import (
	"context"
	"log"

	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/model"
	"github.com/xiangxn/polyman/internal/utils"
)

type LiveExecutor struct {
	submitter      *ConcurrentSubmitter
	listener       ExecutionListener
	tradeMonitor   *TradeMonitor
	balanceManager *BalanceManager
}

func NewLiveExecutor(pmClient *pm.PolymarketClient,
	listener ExecutionListener,
	config config.OrderEngineConfig,
	pmConfig pm.PolymarketConfig,
	balanceConfig config.BalanceConfig,
) *LiveExecutor {
	return &LiveExecutor{
		submitter:      NewConcurrentSubmitter(pmClient, config),
		tradeMonitor:   NewTradeMonitor(pmConfig.ClobWSBaseURL, *pmConfig.FunderAddress, pmConfig.CLOBCreds),
		listener:       listener,
		balanceManager: NewBalanceManager(&balanceConfig),
	}
}

func (e *LiveExecutor) Submit(ctx context.Context, intents []model.Intent) error {
	ins := utils.Filter(intents, func(it model.Intent) bool {
		return it.Side == model.SideBuy
	})
	var amount float64
	for _, in := range ins {
		amount += in.Price * in.Size
	}
	if err := e.balanceManager.Freeze(amount); err != nil {
		log.Printf("[LiveExecutor] Failed to freeze balance: %v", err)
		return err
	}
	return e.submitter.Submit(ctx, intents)
}

func (e *LiveExecutor) Run(ctx context.Context) error {
	log.Printf("[LiveExecutor] Run start")
	e.submitter.SetOnReject(e.OnOrderRejected)
	// 启动订单监听
	go e.tradeMonitor.Run(ctx)
	// 启动全额余额监听
	go e.balanceManager.Run(ctx)
	// 监听订单
	fillCh := e.tradeMonitor.Subscribe()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case fill := <-fillCh:
				e.OnOrderFilled(fill)
			}
		}
	}()
	// 下单
	return e.submitter.Run(ctx)
}

func (e *LiveExecutor) OnOrderFilled(fill model.Fill) {
	if fill.Side == model.SideBuy {
		e.balanceManager.Unfreeze(fill.Price * fill.Size)
	}
	if e.listener != nil {
		e.listener.OnEvent(model.ExecutionEvent{
			Type: model.EventFill,
			Fill: &fill,
		})
	}
}

func (e *LiveExecutor) OnOrderRejected(intent model.Intent, err error) {
	if intent.Side == model.SideBuy {
		e.balanceManager.Unfreeze(intent.Price * intent.Size)
	}
	if e.listener != nil {
		e.listener.OnEvent(model.ExecutionEvent{
			Type:   model.EventReject,
			Intent: &intent,
			Err:    err,
		})
	}
}

func (e *LiveExecutor) SetListener(listener ExecutionListener) {
	e.listener = listener
}
