package marketdata

import (
	"context"

	"github.com/xiangxn/polyman/internal/engine"
	"github.com/xiangxn/polyman/internal/model"

	PM "github.com/xiangxn/go-polymarket-sdk/polymarket"
)

type MarketData interface {
	Run(ctx context.Context) error
	Subscribe() <-chan model.Tick
}

type MarketDataController interface {
	engine.Controller
	SubscribeTokens(tokens ...string)
	UnsubscribeTokens(tokens ...string)

	GetTokenPrice(tokenID string) (PM.PriceData, error)
	GetClient() *PM.PolymarketClient
	Reset()
}
