package strategies

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/model"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

type PolymanStrategy struct {
	MarketSlug string
	mdCtrl     marketdata.MarketDataController
	Tokens     []string
}

func (s *PolymanStrategy) OnTick(t model.Tick) []model.Intent {
	// log.Printf("[PolymanStrategy] tick: %+v, %d", t, len(s.Tokens))
	if len(s.Tokens) < 2 {
		log.Printf("[PolymanStrategy] Tokens length less than 2: %+v", s.Tokens)
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

	if token0.BestAsk.Price+token1.BestAsk.Price < 1.0 { // 价格和小于1是基本条件
		now := time.Now().UnixMilli()
		log.Printf("Book Data === BestAsk: %.2f/%.2f=%.2f, %.2f/%.2f, delay: %d, diff: %d, now: %d", token0.BestAsk.Price, token1.BestAsk.Price, token0.BestAsk.Price+token1.BestAsk.Price, token0.BestAsk.Size, token1.BestAsk.Size, now-t.Timestamp, token0.Timestamp-token1.Timestamp, now)
		if now-t.Timestamp < 500 { // 小于500ms的才尝试下单
			maxSize := 10.0
			minSize := math.Min(token0.BestAsk.Size, token1.BestAsk.Size)
			minSize = minSize * 0.5
			if minSize < 5 {
				return nil
			}
			size := math.Min(minSize, maxSize)
			log.Printf("[PolymanStrategy] Order size: %.2f, token0Price: %.2f, token1Price: %.2f", size, token0.BestAsk.Price, token1.BestAsk.Price)
			return []model.Intent{
				{
					Market: t.Market,
					Token:  token0.TokenID,
					Size:   size,
					Price:  token0.BestAsk.Price,

					Side:      model.SideBuy,
					OrderType: orders.FAK,
				},
				{
					Market: t.Market,
					Token:  token1.TokenID,
					Size:   size,
					Price:  token1.BestAsk.Price,

					Side:      model.SideBuy,
					OrderType: orders.FAK,
				},
			}
		}
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
			utils.SleepWithCtx(ctx, 5*time.Second)
			continue
		}

		endData, err := utils.ToTimestamp(market.Get("endDate").String())
		if err != nil {
			log.Println("ToTimestamp failed:", err)
			utils.SleepWithCtx(ctx, 5*time.Second)
			continue
		}

		var tokenIds []string
		gjson.Parse(market.Get("clobTokenIds").String()).ForEach(func(key, value gjson.Result) bool {
			tokenIds = append(tokenIds, value.String())
			return true
		})
		s.Tokens = tokenIds

		// 先获取TickSize与FeeRateBps
		tickSize := market.Get("orderPriceMinTickSize").Float()
		negRisk := market.Get("negRisk").Bool()
		feesEnabled := market.Get("feesEnabled").Bool()
		for _, tokenId := range s.Tokens {
			pmClient.SetTickSize(tokenId, tickSize)
			pmClient.SetNegRisk(tokenId, negRisk)
			if !feesEnabled {
				pmClient.SetFeeRateBps(tokenId, 0)
			} else {
				pmClient.GetFeeRateBps(tokenId)
			}
		}

		s.mdCtrl.SubscribeTokens(tokenIds...)

		deadline := time.UnixMilli(endData + 1000)
		d := time.Until(deadline)
		if d <= 0 {
			s.mdCtrl.Reset()
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
