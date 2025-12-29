package model

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
