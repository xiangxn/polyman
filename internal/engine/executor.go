package engine

import (
	"context"

	"github.com/xiangxn/polyman/internal/model"
)

type Executor interface {
	Run(ctx context.Context) error
}

type ExecutorController interface {
	Submit(ctx context.Context, intents []model.Intent) error
}
