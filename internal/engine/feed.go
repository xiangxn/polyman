package engine

import "context"

type Feed interface {
	Run(ctx context.Context) error
	SetEventBus(bus *EventBus)
}
