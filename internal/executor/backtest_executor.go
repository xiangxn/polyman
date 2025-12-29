package executor

import "github.com/xiangxn/polyman/internal/model"

type Matcher interface {
	Match(intent model.Intent, tick model.Tick) *model.Fill
}

type BacktestExecutor struct {
	listener ExecutionListener
	matcher  Matcher
}
