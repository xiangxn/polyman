package executor

import (
	"context"
	"log"

	"github.com/xiangxn/polyman/internal/model"
)

type SimpleExecutor struct {
	ch chan model.Intent
}

func NewSimpleExecutor() *SimpleExecutor {
	return &SimpleExecutor{
		ch: make(chan model.Intent, 100),
	}
}

func (e *SimpleExecutor) Submit(ctx context.Context, intent model.Intent) error {
	select {
	case e.ch <- intent:
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
