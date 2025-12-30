package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xiangxn/go-polymarket-sdk/headers"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	"github.com/xiangxn/go-polymarket-sdk/utils"
	"github.com/xiangxn/polyman/internal/model"
)

type TradeMonitor struct {
	ws             *utils.WebSocketClient
	creds          *headers.ApiKeyCreds
	funderAddress  string
	clobUserWSSURL string

	fillCh chan model.Fill
}

func NewTradeMonitor(wsBaseUrl string, funderAddress string, creds *headers.ApiKeyCreds) *TradeMonitor {
	return &TradeMonitor{
		creds:          creds,
		clobUserWSSURL: fmt.Sprintf("%s/ws/user", wsBaseUrl),
		funderAddress:  funderAddress,
		fillCh:         make(chan model.Fill, 4096),
	}
}

func (tm *TradeMonitor) Run(ctx context.Context) error {
	if tm.ws != nil && tm.ws.IsAlive() {
		return nil
	}
	tm.ws = utils.NewWebSocketClient(tm.clobUserWSSURL, 10*time.Second)

	tm.ws.On("open", func(_ any) {
		log.Println("[TradeMonitor] WebSocket Connected")
		tm.subscribeUserTrade()
	})
	tm.ws.On("error", func(e any) {
		log.Println("[TradeMonitor] WebSocket Error:", e)
	})
	tm.ws.On("close", func(_ any) {
		log.Println("[TradeMonitor] WebSocket Closed")
	})
	tm.ws.OnMessage(func(msg []byte) {
		if string(msg) != "PONG" {
			tm.handleMessage(ctx, msg)
		}
	})

	tm.ws.Start()

	// 等待 ctx 结束
	<-ctx.Done()
	return ctx.Err()
}

func (tm *TradeMonitor) subscribeUserTrade() {
	if tm.ws != nil && tm.ws.IsAlive() {
		return
	}
	subscribeMessage := model.SubscribeUserMessage{
		Type:    "USER",
		Markets: []string{},
		Auth:    tm.getAuth(),
	}

	data, _ := json.Marshal(subscribeMessage)
	err := tm.ws.Send(data)
	if err != nil {
		log.Printf("订阅User交易失败: %v", err)
		return
	}
}

func (tm *TradeMonitor) getAuth() model.ClobAuth {
	if tm.creds == nil {
		return model.ClobAuth{}
	}

	return model.ClobAuth{
		ApiKey:     tm.creds.Key,
		Secret:     tm.creds.Secret,
		Passphrase: tm.creds.Passphrase,
	}
}

func (tm *TradeMonitor) handleMessage(ctx context.Context, msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TradeMonitor] handleMessage panic: %v", r)
		}
	}()

	eventType := gjson.Get(string(msg), "event_type").String()
	if eventType != "trade" && eventType != "order" {
		return
	}

	if eventType == "trade" {
		var wsTrade model.WSTrade
		err := json.Unmarshal([]byte(msg), &wsTrade)
		if err != nil {
			log.Printf("[TradeMonitor] handleMessage json.Unmarshal error: %v", err)
			return
		}
		if wsTrade.Status == "" {

		}
		side := model.SideBuy
		fill := model.Fill{
			FillID:   wsTrade.Id,
			MarketID: wsTrade.Market,
			Time:     wsTrade.Matchtime,
		}
		if wsTrade.TradeOwner == tm.creds.Key { // taker
			if wsTrade.Side == string(orders.SELL) {
				side = model.SideSell
			}
			fill.OrderID = wsTrade.TakerOrderId
			fill.TokenID = wsTrade.AssetId
			fill.Side = side
			fill.Price = wsTrade.Price
			fill.Size = wsTrade.Size
			fill.Fee = wsTrade.FeeRateBps * wsTrade.Size * wsTrade.Price
			// 异步发送到 channel
			select {
			case tm.fillCh <- fill:
			case <-ctx.Done():
				log.Println("[TradeMonitor] handleMessage taker ctx.Done")
			}
		} else { // maker
			for _, mo := range wsTrade.MakerOrders {
				if mo.Owner == tm.creds.Key {
					if mo.Side == string(orders.SELL) {
						side = model.SideSell
					}
					newFill := fill
					newFill.OrderID = mo.OrderId
					newFill.TokenID = mo.AssetId
					newFill.Side = side
					newFill.Price = mo.Price
					newFill.Size = mo.MatchedAmount
					newFill.Fee = mo.FeeRateBps * mo.MatchedAmount * mo.Price
					// 异步发送到 channel
					select {
					case tm.fillCh <- newFill:
					case <-ctx.Done():
						log.Println("[TradeMonitor] handleMessage maker ctx.Done")
					}
				}
			}
		}
		return
	}

	// 选择用trade同步position，不用order, TODO: 后面可以处理取消订单
	// if eventType == "order" {
	// 	var wsOrder model.WSOrder
	// 	err := json.Unmarshal([]byte(msg), &wsOrder)
	// 	if err != nil {
	// 		log.Printf("[TradeMonitor] handleMessage json.Unmarshal error: %v", err)
	// 		return
	// 	}
	// 	fill := model.Fill{}
	// 	select {
	// 	case tm.fillCh <- fill:
	// 	case <-ctx.Done():
	// 		log.Println("[TradeMonitor] handleMessage ctx.Done")
	// 	}
	// 	return
	// }
}

func (tm *TradeMonitor) Subscribe() <-chan model.Fill {
	return tm.fillCh
}
