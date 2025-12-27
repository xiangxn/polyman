package strategies

import (
	"context"
	"polyman/internal/model"
)

// 核心 Tick 处理
type Strategy interface {
	OnTick(t model.Tick) []model.Intent
}

// 初始化方法，可以加载配置、缓存、订阅事件等
type InitnableStrategy interface {
	Init(ctx context.Context) error
}

// 可选的周期性处理，或市场结束处理
type RunnableStrategy interface {
	Run(ctx context.Context) error
}
