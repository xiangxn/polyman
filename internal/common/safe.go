package common

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

type SafeGroup struct {
	ctx    context.Context
	cancel context.CancelFunc

	wg  sync.WaitGroup
	err error
	mu  sync.Mutex
}

func NewSafeGroup(parent context.Context) *SafeGroup {
	ctx, cancel := context.WithCancel(parent)
	return &SafeGroup{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (g *SafeGroup) Context() context.Context {
	return g.ctx
}

func (g *SafeGroup) Go(name string, fn func(ctx context.Context) error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] %s: %v\n%s", name, r, debug.Stack())
				g.setError(panicToError(r))
			}
		}()

		if err := fn(g.ctx); err != nil {
			g.setError(err)
		}
	}()
}

func (g *SafeGroup) setError(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.err == nil {
		g.err = err
		g.cancel()
	}
}

func (g *SafeGroup) Wait() error {
	g.wg.Wait()
	return g.err
}

func panicToError(r any) error {
	return fmt.Errorf("panic: %v", r)
}
