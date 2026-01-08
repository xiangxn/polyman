package executor

import (
	"context"

	"github.com/xiangxn/polyman/pkg/model"
)

type Executor interface {
	Submit(ctx context.Context, intents []model.Intent) error
	Run(ctx context.Context) error
}

type ExecutionListener interface {
	OnEvent(evt model.ExecutionEvent)
}

type EventSource interface {
	SetListener(listener ExecutionListener)
}
