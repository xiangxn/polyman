package executor

import "github.com/xiangxn/polyman/internal/model"

type EventType int

const (
	EventFill EventType = iota
	EventReject
	EventCancel
	EventExpire
)

type ExecutionEvent struct {
	Type EventType

	Fill   *model.Fill
	Intent *model.Intent
	Err    error
}
