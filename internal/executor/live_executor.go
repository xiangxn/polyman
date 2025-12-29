package executor

import (
	"context"

	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/model"
)

type LiveExecutor struct {
	submitter *ConcurrentSubmitter
	listener  ExecutionListener
}

func NewLiveExecutor(pmClient *pm.PolymarketClient, listener ExecutionListener, config config.OrderEngineConfig) *LiveExecutor {
	return &LiveExecutor{
		submitter: NewConcurrentSubmitter(pmClient, config),
		listener:  listener,
	}
}

func (e *LiveExecutor) Submit(ctx context.Context, intent model.Intent) error {
	return e.submitter.Submit(ctx, intent)
}

func (e *LiveExecutor) Run(ctx context.Context) error {
	// TODO: 实现监听订单
	return e.submitter.Run(ctx)
}

func (e *LiveExecutor) OnOrderFilled(fill model.Fill) {
	key := OrderKey{Token: fill.TokenID, Side: fill.Side}

	// 1️⃣ 释放去重 cache（事实层）
	e.submitter.cache.Delete(key)

	// 2️⃣ 发出统一事件
	if e.listener != nil {
		e.listener.OnEvent(model.ExecutionEvent{
			Type: model.EventFill,
			Fill: &fill,
		})
	}
}

func (e *LiveExecutor) OnOrderRejected(intent model.Intent, err error) {
	key := OrderKey{Token: intent.Token, Side: intent.Side}

	e.submitter.cache.Delete(key)

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
