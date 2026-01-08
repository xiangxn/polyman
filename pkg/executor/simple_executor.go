package executor

import (
	"context"
	"log"

	"github.com/xiangxn/polyman/pkg/engine"
)

type SimpleExecutor struct {
	ch chan []engine.Intent
}

func NewSimpleExecutor() *SimpleExecutor {
	return &SimpleExecutor{
		ch: make(chan []engine.Intent, 100),
	}
}

func (e *SimpleExecutor) Submit(ctx context.Context, intents []engine.Intent) error {
	select {
	case e.ch <- intents:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *SimpleExecutor) Run(ctx context.Context) error {
	for {
		select {
		case in := <-e.ch:
			log.Printf("[Executor] ORDER: %+v\n", in)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
