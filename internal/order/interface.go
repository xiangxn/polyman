package order

import (
	"context"
	"polyman/internal/model"
)

type Executor interface {
	Submit(ctx context.Context, intent model.Intent) error
	Run(ctx context.Context) error
}
