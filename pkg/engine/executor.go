package engine

import (
	"context"
)

type Executor interface {
	Run(ctx context.Context) error
}

type ExecutorController interface {
	Submit(ctx context.Context, intents []Intent) error
}
