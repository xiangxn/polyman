package model

import "sync"

type EventType int

const (
	EventFill EventType = iota
	EventReject
	EventCancel
	EventExpire
)

type ExecutionEvent struct {
	Type EventType

	Fill   *Fill
	Intent *Intent
	Err    error
}

type OrderCache struct {
	mu           sync.Mutex
	OriginalSize float64
	FilledSize   float64
}
