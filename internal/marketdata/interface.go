package marketdata

import (
	"context"
	"polyman/internal/model"
)

type MarketData interface {
	Run(ctx context.Context) error
	Subscribe() <-chan model.Tick
}
