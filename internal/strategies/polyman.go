package strategies

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/model"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

type PolymanStrategy struct {
	MarketSlug string
	mdCtrl     marketdata.MarketDataController
	Tokens     []string
}

func (s *PolymanStrategy) OnTick(t model.Tick) []model.Intent {
	if len(s.Tokens) < 2 {
		log.Printf("Tokens length less than 2: %+v", s.Tokens)
		return nil
	}
	token0, err := s.mdCtrl.GetTokenPrice(s.Tokens[0])
	if err != nil {
		return nil
	}
	token1, err := s.mdCtrl.GetTokenPrice(s.Tokens[1])
	if err != nil {
		return nil
	}

	if token0.BestAsk.Price == 0 || token1.BestAsk.Price == 0 {
		return nil
	}

	if token0.BestAsk.Price+token1.BestAsk.Price < 1.0 {
		now := time.Now().UnixMilli()
		log.Printf("Book Data === BestAsk: %.2f/%.2f=%.2f, %.2f/%.2f, delay: %d, diff: %d", token0.BestAsk.Price, token1.BestAsk.Price, token0.BestAsk.Price+token1.BestAsk.Price, token0.BestAsk.Size, token1.BestAsk.Size, now-t.Timestamp, token0.Timestamp-token1.Timestamp)
	}

	return nil
}

func (s *PolymanStrategy) Init(ctx context.Context, ctrl marketdata.MarketDataController) error {
	s.mdCtrl = ctrl
	return nil
}

// 可选的周期性处理，或市场结束处理
func (s *PolymanStrategy) Run(ctx context.Context) error {
	pmClient := s.mdCtrl.GetClient()
	for {
		marketSlug := fmt.Sprintf("%s%d", s.MarketSlug, utils.RoundTo15Minutes())
		market, err := pmClient.FetchMarketBySlug(marketSlug)
		if err != nil {
			log.Println("FetchMarketBySlug failed:", err)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		endData, err := utils.ToTimestamp(market.Get("endDate").String())
		if err != nil {
			log.Println("ToTimestamp failed:", err)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		var tokenIds []string
		gjson.Parse(market.Get("clobTokenIds").String()).ForEach(func(key, value gjson.Result) bool {
			tokenIds = append(tokenIds, value.String())
			return true
		})
		s.Tokens = tokenIds

		s.mdCtrl.SubscribeTokens(tokenIds...)

		deadline := time.UnixMilli(endData + 1000)
		d := time.Until(deadline)
		if d <= 0 {
			continue // 直接下一轮
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			s.mdCtrl.Reset()
		}
	}
}
