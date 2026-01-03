package strategies

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/xiangxn/polyman/internal/marketdata"
	"github.com/xiangxn/polyman/internal/model"
	pmHelper "github.com/xiangxn/polyman/internal/utils"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

const eps = 1e-9

type PolyeventStrategy struct {
	EventSlug string
	mdCtrl    marketdata.MarketDataController
	Tokens    []string
	event     PolyEvent

	print        map[string]bool
	printNegRisk bool
	lastTime     int64
}

type PolyEvent struct {
	ID      string
	Markets map[string]PolyMarket
	NegRisk bool
}

type PolyMarket struct {
	ID          string
	ConditionId string
	Tokens      []string
}

func (s *PolyeventStrategy) OnTick(t model.Tick) []model.Intent {
	// log.Printf("[PolyeventStrategy] tick: %+v, %d", t, len(s.Tokens))
	length := len(s.Tokens)
	if length < 2 {
		log.Printf("[PolyeventStrategy] Tokens length less than 2: %+v", s.Tokens)
		return nil
	}

	tokens := make(map[string]polymarket.PriceData)

	for _, tid := range s.Tokens {
		t, err := s.mdCtrl.GetTokenPrice(tid)
		if err != nil {
			log.Printf("[PolyeventStrategy] Error getting token price: %v", err)
			return nil
		} else {
			if t.BestAsk.Price == 0 {
				return nil
			}
			tokens[tid] = t
		}
	}

	marketTotalPrice := 0.0
	pStr := ""
	sStr := ""
	market := s.event.Markets[t.Market]
	for i, tid := range market.Tokens {
		marketTotalPrice += tokens[tid].BestAsk.Price
		if i == len(market.Tokens)-1 {
			pStr += fmt.Sprintf("%.2f", tokens[tid].BestAsk.Price)
			sStr += fmt.Sprintf("%.2f", tokens[tid].BestAsk.Size)
		} else {
			pStr += fmt.Sprintf("%.2f+", tokens[tid].BestAsk.Price)
			sStr += fmt.Sprintf("%.2f/", tokens[tid].BestAsk.Size)
		}
	}
	if s.print[t.Market] {
		s.print[t.Market] = false
		now := time.Now().UnixMilli()
		log.Printf("Market[%s] === Prices: %s=%.2f, Sizes: %s, MS: %d, Now: %d", market.ID, pStr, marketTotalPrice, sStr, now-s.lastTime, now)
	}
	if !(math.Abs(marketTotalPrice-1.0) < eps) && marketTotalPrice < 1.0 { // 价格和小于1是基本条件
		s.print[t.Market] = true
		now := time.Now().UnixMilli()
		s.lastTime = now
		log.Printf("Market[%s] === Prices: %s=%.2f, Sizes: %s, Delay: %d, Now: %d", market.ID, pStr, marketTotalPrice, sStr, now-t.Timestamp, now)
	}

	if s.event.NegRisk {
		pStr := ""
		sStr := ""
		negriskTotalPrice := 0.0
		var tokenIds []string
		for _, v := range s.event.Markets {
			tokenIds = append(tokenIds, v.Tokens[0])
		}
		length = len(tokenIds)
		for i, t := range tokenIds {
			negriskTotalPrice += tokens[t].BestAsk.Price
			if i == length-1 {
				pStr += fmt.Sprintf("%.2f", tokens[t].BestAsk.Price)
				sStr += fmt.Sprintf("%.2f", tokens[t].BestAsk.Size)
			} else {
				pStr += fmt.Sprintf("%.2f+", tokens[t].BestAsk.Price)
				sStr += fmt.Sprintf("%.2f/", tokens[t].BestAsk.Size)
			}
		}
		if s.printNegRisk {
			s.printNegRisk = false
			now := time.Now().UnixMilli()
			log.Printf("Event[%s] === Prices: %s=%.2f, Sizes: %s, MS: %d, Now: %d", s.event.ID, pStr, negriskTotalPrice, sStr, now-s.lastTime, now)
		}

		if !(math.Abs(negriskTotalPrice-1.0) < eps) && negriskTotalPrice < 1.0 { // 价格和小于1是基本条件
			s.printNegRisk = true
			now := time.Now().UnixMilli()
			s.lastTime = now
			log.Printf("Event[%s] === Prices: %s=%.2f, Sizes: %s, Delay: %d, Now: %d", s.event.ID, pStr, negriskTotalPrice, sStr, now-t.Timestamp, now)
		}
	}

	return nil
}

func (s *PolyeventStrategy) Init(ctx context.Context, ctrl marketdata.MarketDataController) error {
	s.mdCtrl = ctrl
	s.print = make(map[string]bool)
	return nil
}

// 可选的周期性处理，或市场结束处理
func (s *PolyeventStrategy) Run(ctx context.Context) error {
	log.Println("[PolyeventStrategy] Run start")
	defer log.Println("[PolyeventStrategy] Run exit")

	pmClient := s.mdCtrl.GetClient()
	for {
		eventSlug := pmHelper.FormatSlug(s.EventSlug)
		event, err := pmClient.FetchEventBySlug(eventSlug)
		if err != nil {
			log.Println("FetchEventBySlug failed:", err)
			utils.SleepWithCtx(ctx, 5*time.Second)
			continue
		}

		endDateStr := event.Get("endDate").String()
		endDate, err := utils.ToTimestamp(endDateStr)
		if err != nil {
			log.Println("ToTimestamp failed:", err)
			utils.SleepWithCtx(ctx, 5*time.Second)
			continue
		}

		s.event = PolyEvent{
			ID:      event.Get("id").String(),
			NegRisk: event.Get("negRisk").Bool(),
			Markets: make(map[string]PolyMarket),
		}
		log.Printf("[PolyeventStrategy] Event: %s, NegRisk: %t, EndData: %s", s.event.ID, s.event.NegRisk, endDateStr)

		var eventTokens []string
		for _, market := range event.Get("markets").Array() {
			var marketTokens []string
			gjson.Parse(market.Get("clobTokenIds").String()).ForEach(func(key, value gjson.Result) bool {
				eventTokens = append(eventTokens, value.String())
				marketTokens = append(marketTokens, value.String())
				return true
			})

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
			conditionId := market.Get("conditionId").String()
			s.event.Markets[conditionId] = PolyMarket{
				ID:          market.Get("id").String(),
				ConditionId: conditionId,
				Tokens:      marketTokens,
			}
		}
		s.Tokens = eventTokens

		s.mdCtrl.SubscribeTokens(eventTokens...)

		if s.event.NegRisk {
			closed := event.Get("closed").Bool()
			active := event.Get("active").Bool()
			if closed && !active {
				return ctx.Err() // 直接退出
			}
			timer := time.NewTicker(1 * time.Minute)
			defer timer.Stop()
			for { // 循环检查event是否关闭
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-timer.C: // 1分钟更新一下event的数据,但是不用重连接
					newE, err := pmClient.FetchEventBySlug(eventSlug)
					if err == nil {
						closed = newE.Get("closed").Bool()
						active = newE.Get("active").Bool()
						log.Printf("[PolyeventStrategy] Event: %s, closed: %v, active: %v", s.event.ID, closed, active)
					}
					if closed && !active {
						return ctx.Err() // 直接退出
					}
				}
			}
		} else {
			deadline := time.UnixMilli(endDate + 1000)
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
}
