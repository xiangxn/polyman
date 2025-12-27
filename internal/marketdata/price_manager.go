package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"polyman/internal/model"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	PM "github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/go-polymarket-sdk/utils"
)

type PriceManager struct {
	ws               *utils.WebSocketClient
	tickCh           chan model.Tick
	mu               sync.RWMutex
	clobMarketWSSURL string
	isConnecting     bool
	subsTokens       []string
}

func NewPriceManager(wsBaseUrl string) *PriceManager {
	return &PriceManager{
		tickCh:           make(chan model.Tick, 1024), // channel buffer
		clobMarketWSSURL: fmt.Sprintf("%s/ws/market", wsBaseUrl),
	}
}

// Run 启动 WebSocket 监听
func (pm *PriceManager) Run(ctx context.Context) error {
	pm.mu.Lock()
	if pm.isConnecting || pm.ws != nil {
		pm.mu.Unlock()
		return nil
	}
	pm.isConnecting = true
	pm.mu.Unlock()

	pm.ws = utils.NewWebSocketClient(pm.clobMarketWSSURL, 10*time.Second)

	pm.ws.On("open", func(_ any) {
		log.Println("[PriceManager] WebSocket Connected")
		pm.isConnecting = false
		pm.subscribeToMarket()
	})
	pm.ws.On("error", func(e any) {
		log.Println("[PriceManager] WebSocket Error:", e)
		pm.isConnecting = false
	})
	pm.ws.On("close", func(_ any) {
		log.Println("[PriceManager] WebSocket Closed")
		pm.isConnecting = false
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
	close(pm.tickCh)
	return ctx.Err()
}

// Subscribe 返回 channel
func (pm *PriceManager) Subscribe() <-chan model.Tick {
	return pm.tickCh
}

// handleMessage 解析消息并发送 Tick
func (pm *PriceManager) handleMessage(msg string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PriceManager] handleMessage panic: %v", r)
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
		lastBid := Bids[len(Bids)-1]
		bestBid.Price = lastBid.Get("price").Float()
		bestBid.Size = lastBid.Get("size").Float()
	}
	if len(Asks) > 0 {
		lastAsk := Asks[len(Asks)-1]
		bestAsk.Price = lastAsk.Get("price").Float()
		bestAsk.Size = lastAsk.Get("size").Float()
	}

	tick := model.Tick{
		Market:    market,
		Token:     assetID,
		Price:     bestBid.Price, // 可以根据策略选择 BestBid / BestAsk / Mid
		Volume:    bestBid.Size,  // 可选填
		Timestamp: timestamp,
		Extra: map[string]any{
			"best_bid": bestBid,
			"best_ask": bestAsk,
		},
	}

	// 异步发送到 channel
	select {
	case pm.tickCh <- tick:
	default:
		// channel 满了可以丢弃或打 log
		log.Println("[PriceManager] tick channel full, dropping tick")
	}
}

// Disconnect 断开 WS
func (pm *PriceManager) Disconnect() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.ws != nil {
		pm.ws.Close()
		pm.ws = nil
	}
	pm.isConnecting = false
	pm.subsTokens = nil
}

// SubscribeToMarket 订阅市场数据 (导出的方法)
func (pm *PriceManager) SubscribeToMarket(tokens ...string) {
	pm.subscribeToMarket(tokens...)
}

func (pm *PriceManager) subscribeToMarket(tokens ...string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

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

	if len(pm.subsTokens) == 0 || pm.ws == nil {
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
