package executor

import (
	"context"

	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/model"
)

type LiveExecutor struct {
	submitter    *ConcurrentSubmitter
	listener     ExecutionListener
	tradeMonitor *TradeMonitor
}

func NewLiveExecutor(pmClient *pm.PolymarketClient, listener ExecutionListener, config config.OrderEngineConfig, pmConfig pm.PolymarketConfig) *LiveExecutor {
	return &LiveExecutor{
		submitter:    NewConcurrentSubmitter(pmClient, config),
		tradeMonitor: NewTradeMonitor(pmConfig.ClobWSBaseURL, *pmConfig.FunderAddress, pmConfig.CLOBCreds),
		listener:     listener,
	}
}

func (e *LiveExecutor) Submit(ctx context.Context, intent model.Intent) error {
	return e.submitter.Submit(ctx, intent)
}

func (e *LiveExecutor) Run(ctx context.Context) error {
	e.submitter.SetOnReject(e.OnOrderRejected)
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
	if e.listener != nil {
		e.listener.OnEvent(model.ExecutionEvent{
			Type: model.EventFill,
			Fill: &fill,
		})
	}
}

func (e *LiveExecutor) OnOrderRejected(intent model.Intent, err error) {
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
