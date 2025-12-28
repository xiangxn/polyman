package order

import (
	"context"

	"github.com/xiangxn/polyman/internal/model"
)

type Executor interface {
	Submit(ctx context.Context, intent model.Intent) error
	Run(ctx context.Context) error
}
