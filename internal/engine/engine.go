package engine

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type Engine[C Controller] struct {
	bus         *EventBus
	feeds       []Feed
	strategies  []Strategy[C]
	executor    Executor
	controllers map[string]C
}

func NewEngine[C Controller](
	feeds []Feed,
	strategies []Strategy[C],
	executor Executor,
) *Engine[C] {
	bus := NewEventBus()
	for _, feed := range feeds {
		feed.SetEventBus(bus)
	}
	return &Engine[C]{
		bus:        bus,
		feeds:      feeds,
		strategies: strategies,
		executor:   executor,
	}
}

func (e *Engine[C]) Run(parent context.Context) error {
	g, engineCtx := errgroup.WithContext(parent)

	// Executor：关键域
	g.Go(func() error {
		return e.executor.Run(engineCtx)
	})

	// Feed：非关键域（可以自己重连）
	for _, f := range e.feeds {
		feed := f
		if c, ok := feed.(C); ok {
			e.controllers[c.Name()] = c
		}
		g.Go(func() error {
			return feed.Run(engineCtx)
		})
	}

	// Strategy：半关键域
	for _, s := range e.strategies {
		strat := s
		strat.SetEventBus(e.bus)
		g.Go(func() error {
			return strat.Run(engineCtx, e.executor.(ExecutorController), e.controllers)
		})
	}

	return g.Wait()
}
