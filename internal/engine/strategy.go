package engine

import "context"

type Strategy[C Controller] interface {
	Name() string
	Run(ctx context.Context, executor ExecutorController, dataCtrls map[string]C) error
	SetEventBus(bus *EventBus)
}
