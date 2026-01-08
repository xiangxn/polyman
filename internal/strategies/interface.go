package strategies

import (
	"context"

	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/model"
)

type Strategy interface {
	ID() string
	OnEvent(ctx context.Context, ev model.MarketEvent) ([]model.Intent, error)
}

// 初始化方法，可以加载配置、缓存、订阅事件等
type InitnableStrategy interface {
	Init(ctx context.Context, ctrl marketdata.MarketDataController) error
}

// 可选的周期性处理，或市场结束处理
type RunnableStrategy interface {
	Run(ctx context.Context) error
}
