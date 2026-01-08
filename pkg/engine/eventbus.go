package engine

import (
	"sync"
	"sync/atomic"
)

type EventBus struct {
	mu     sync.RWMutex
	hubs   map[EventType]*eventHub
	closed atomic.Bool
}

type eventHub struct {
	subs   map[uint64]chan Event
	nextID uint64
}

func NewEventBus() *EventBus {
	return &EventBus{
		hubs: make(map[EventType]*eventHub),
	}
}

type Subscription struct {
	C      <-chan Event
	cancel func()
}

func (b *EventBus) Subscribe(et EventType, buf int) Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	hub, ok := b.hubs[et]
	if !ok {
		hub = &eventHub{subs: make(map[uint64]chan Event)}
		b.hubs[et] = hub
	}

	id := hub.nextID
	hub.nextID++

	ch := make(chan Event, buf)
	hub.subs[id] = ch

	return Subscription{
		C: ch,
		cancel: func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			delete(hub.subs, id)
			close(ch)
		},
	}
}

func (b *EventBus) Publish(e Event) {
	if b.closed.Load() {
		return
	}

	b.mu.RLock()
	hub := b.hubs[e.Type()]
	b.mu.RUnlock()

	if hub == nil {
		return
	}

	for _, ch := range hub.subs {
		select {
		case ch <- e:
		default:
			// drop
		}
	}
}

func SubscribeTyped[T Event](
	bus *EventBus,
	et EventType,
	buf int,
) (<-chan T, func()) {

	sub := bus.Subscribe(et, buf)
	out := make(chan T, buf)

	go func() {
		defer close(out)
		for e := range sub.C {
			out <- e.(T)
		}
	}()

	return out, sub.cancel
}
