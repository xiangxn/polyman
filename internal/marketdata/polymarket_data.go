package marketdata

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xiangxn/polyman/internal/model"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	PM "github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

type PolymarketData struct {
	ws               *utils.WebSocketClient
	tickCh           chan model.Tick
	tokensPrice      map[string]*PM.PriceData
	mu               sync.RWMutex
	clobMarketWSSURL string
	isConnecting     atomic.Bool
	subsTokens       []string
	muSubsTokens     sync.RWMutex
	pmClient         *PM.PolymarketClient
}

func NewPolymarketData(wsBaseUrl string, client *PM.PolymarketClient) *PolymarketData {
	return &PolymarketData{
		tickCh:           make(chan model.Tick, 1024), // channel buffer
		tokensPrice:      make(map[string]*PM.PriceData),
		clobMarketWSSURL: fmt.Sprintf("%s/ws/market", wsBaseUrl),
		pmClient:         client,
	}
}

func (pm *PolymarketData) GetClient() *PM.PolymarketClient {
	return pm.pmClient
}

// Run 启动 WebSocket 监听
func (pm *PolymarketData) Run(ctx context.Context) error {
	if pm.isConnecting.Load() || (pm.ws != nil && pm.ws.IsAlive()) {
		return nil
	}

	pm.isConnecting.Store(true)

	pm.ws = utils.NewWebSocketClient(pm.clobMarketWSSURL, 10*time.Second)

	pm.ws.On("open", func(_ any) {
		log.Println("[PolymarketData] WebSocket Connected")
		pm.isConnecting.Store(false)
		pm.subscribeToMarket()
	})
	pm.ws.On("error", func(e any) {
		log.Println("[PolymarketData] WebSocket Error:", e)
		pm.isConnecting.Store(false)
	})
	pm.ws.On("close", func(_ any) {
		log.Println("[PolymarketData] WebSocket Closed")
		pm.isConnecting.Store(false)
	})
	pm.ws.On("reconnect", func(_ any) {
		// 清空数据，防止旧数据异常
		pm.mu.Lock()
		pm.tokensPrice = make(map[string]*PM.PriceData)
		pm.mu.Unlock()
	})

	pm.ws.OnMessage(func(msg []byte) {
		if string(msg) != "PONG" {
			pm.handleMessage(string(msg))
		}
	})

	pm.ws.Start()

	// 等待 ctx 结束
	<-ctx.Done()
	pm.Disconnect()
	// close(pm.tickCh)
	return ctx.Err()
}

// Subscribe 返回 channel
func (pm *PolymarketData) Subscribe() <-chan model.Tick {
	return pm.tickCh
}

// handleMessage 解析消息并发送 Tick
func (pm *PolymarketData) handleMessage(msg string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PolymarketData] handleMessage panic: %v", r)
		}
	}()

	eventType := gjson.Get(msg, "event_type").String()
	if eventType != "book" {
		return
	}

	market := gjson.Get(msg, "market").String()
	Bids := gjson.Get(msg, "bids").Array()
	Asks := gjson.Get(msg, "asks").Array()
	assetID := gjson.Get(msg, "asset_id").String()
	timestamp := gjson.Get(msg, "timestamp").Int()

	if len(Bids) == 0 && len(Asks) == 0 {
		return
	}

	var bestBid, bestAsk orders.Book
	if len(Bids) > 0 {
		// lastBid := Bids[len(Bids)-1]
		lastBid := slices.MaxFunc(Bids, func(a, b gjson.Result) int { return cmp.Compare(a.Get("price").Float(), b.Get("price").Float()) })
		bestBid.Price = lastBid.Get("price").Float()
		bestBid.Size = lastBid.Get("size").Float()
	}
	if len(Asks) > 0 {
		// lastAsk := Asks[len(Asks)-1]
		lastAsk := slices.MinFunc(Asks, func(a, b gjson.Result) int { return cmp.Compare(a.Get("price").Float(), b.Get("price").Float()) })
		bestAsk.Price = lastAsk.Get("price").Float()
		bestAsk.Size = lastAsk.Get("size").Float()
	}
	pm.updatePrice(&PM.PriceData{
		TokenID:   assetID,
		BestAsk:   &bestAsk,
		BestBid:   &bestBid,
		Market:    market,
		Timestamp: timestamp,
	})

	tick := model.Tick{
		Market:    market,
		Token:     assetID,
		Price:     bestBid.Price, // 可以根据策略选择 BestBid / BestAsk / Mid
		Volume:    bestBid.Size,  // 可选填
		Timestamp: timestamp,
	}

	// 异步发送到 channel
	select {
	case pm.tickCh <- tick:
	default:
		// channel 满了可以丢弃或打 log
		log.Println("[PolymarketData] tick channel full, dropping tick")
	}
}

// Disconnect 断开 WS
func (pm *PolymarketData) Disconnect() {
	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if pm.ws != nil {
		pm.ws.Close()
		pm.ws = nil
	}
	pm.isConnecting.Store(false)
	pm.subsTokens = nil
}

func (pm *PolymarketData) Reset() {
	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	pm.subsTokens = nil
	pm.tokensPrice = make(map[string]*PM.PriceData)

	if pm.ws != nil {
		pm.ws.Reset()
	}
}

// SubscribeTokens 订阅市场数据 (导出的方法)
func (pm *PolymarketData) SubscribeTokens(tokens ...string) {
	pm.subscribeToMarket(tokens...)
}

func (pm *PolymarketData) UnsubscribeTokens(tokens ...string) {
	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if len(tokens) > 0 {
		for _, token := range tokens {
			// 从订阅列表中移除
			for i, t := range pm.subsTokens {
				if t == token {
					pm.subsTokens = append(pm.subsTokens[:i], pm.subsTokens[i+1:]...)
					break
				}
			}
		}
	}
}

func (pm *PolymarketData) subscribeToMarket(tokens ...string) {
	pm.muSubsTokens.Lock()
	defer pm.muSubsTokens.Unlock()

	if len(tokens) > 0 {
		pm.subsTokens = append(pm.subsTokens, tokens...)
		// 去重
		tokenSet := make(map[string]bool)
		for _, token := range pm.subsTokens {
			tokenSet[token] = true
		}
		pm.subsTokens = make([]string, 0, len(tokenSet))
		for token := range tokenSet {
			pm.subsTokens = append(pm.subsTokens, token)
		}
	}

	if len(pm.subsTokens) == 0 || pm.ws == nil || !pm.ws.IsAlive() {
		return
	}

	subscribeMessage := PM.MarketMessage{
		Type:      "MARKET",
		AssetsIDs: pm.subsTokens,
	}

	data, _ := json.Marshal(subscribeMessage)
	err := pm.ws.Send(data)
	if err != nil {
		log.Printf("订阅市场失败: %v", err)
		return
	}

	log.Printf("📡 已订阅市场: %v", pm.subsTokens)
}

func (pm *PolymarketData) updatePrice(priceData *PM.PriceData) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.tokensPrice[priceData.TokenID] = priceData
}

func (pm *PolymarketData) GetTokenPrice(tokenID string) (PM.PriceData, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if priceData, ok := pm.tokensPrice[tokenID]; ok {
		return *priceData, nil
	}
	return PM.PriceData{}, fmt.Errorf("token price not found for %s", tokenID)
}
