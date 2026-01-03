package executor

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	pmModel "github.com/polymarket/go-order-utils/pkg/model"
	"github.com/xiangxn/go-polymarket-sdk/orders"
	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/model"
)

type ConcurrentSubmitter struct {
	ch      chan []model.Intent
	workers int
	wg      sync.WaitGroup

	pmClient *pm.PolymarketClient

	onSubmit func(intent model.Intent, orderID string)
	onReject func(intent model.Intent, err error)
}

type OrderKey struct {
	Token string
	Side  model.Side
}

func NewConcurrentSubmitter(pmClient *pm.PolymarketClient, config config.OrderEngineConfig) *ConcurrentSubmitter {
	return &ConcurrentSubmitter{
		ch:       make(chan []model.Intent, config.QueueSize),
		workers:  config.WorkerNum,
		pmClient: pmClient,
	}
}

func (e *ConcurrentSubmitter) SetOnSubmit(onSubmit func(intent model.Intent, orderID string)) {
	e.onSubmit = onSubmit
}

func (e *ConcurrentSubmitter) SetOnReject(onReject func(intent model.Intent, err error)) {
	e.onReject = onReject
}

// Submit 投递订单
func (e *ConcurrentSubmitter) Submit(ctx context.Context, intents []model.Intent) error {
	select {
	case e.ch <- intents:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run 启动 worker goroutine
func (e *ConcurrentSubmitter) Run(ctx context.Context) error {
	log.Println("[Executor] Run start")
	defer log.Println("[Executor] Run exit")

	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go func(id int) {
			defer e.wg.Done()
			log.Printf("[Executor] worker-%d 开始", id)
			for {
				select {
				case <-ctx.Done():
					log.Printf("[Executor] worker-%d 停止", id)
					return
				case ins := <-e.ch:
					e.handleOrder(id, ins)
				}
			}
		}(i)
	}

	// 等待所有 worker 停止
	e.wg.Wait()
	return ctx.Err()
}

// handleOrder 处理订单逻辑（打印 / 调用下单）
func (s *ConcurrentSubmitter) handleOrder(workerID int, intents []model.Intent) {
	// ⚠️ 如果这里有共享状态，比如 position，需要加锁
	// log.Printf("[Executor] worker-%d processing order: %+v", workerID, intent)

	// // 下单逻辑
	// if e.pmClient != nil {
	// 	// 创建订单
	// 	side := pmModel.BUY
	// 	if intent.Side == orders.SELL {
	// 		side = pmModel.SELL
	// 	}
	// 	order, err := e.pmClient.CreateOrder(&orders.UserOrder{
	// 		TokenID: intent.Token,
	// 		Price:   intent.Price,
	// 		Size:    intent.Size,
	// 		Side:    side,
	// 	}, orders.CreateOrderOptions{
	// 		TickSize:      orders.TickSize001,
	// 		SignatureType: pmModel.POLY_GNOSIS_SAFE,
	// 	})
	// 	if err != nil {
	// 		log.Printf("[Worker %d] 下单失败: %v", workerID, err)
	// 		return
	// 	}
	// 	// 发送订单
	// 	result, err := e.pmClient.PostOrder(order, intent.OrderType, false)
	// 	if err != nil {
	// 		log.Printf("[Worker %d] 下单失败: %v", workerID, err)
	// 		return
	// 	}
	// 	orderId := result.Get("orderId").String()
	// 	log.Printf("[Worker %d] 下单成功: %v", workerID, orderId)
	// } else {
	// 	log.Printf("[Worker %d] 下单失败: %v", workerID, "pmClient is nil")
	// }

	// 下单完成后释放缓存
	// e.cache.Delete(key)

	count := len(intents)
	if count < 1 {
		return
	}
	// 下单逻辑
	if s.pmClient != nil {
		// 创建订单
		if count == 1 {
			intent := intents[0]
			side := pmModel.BUY
			if intent.Side == model.SideSell {
				side = pmModel.SELL
			}
			signatureType := pmModel.POLY_GNOSIS_SAFE
			order, err := s.pmClient.CreateOrder(&orders.UserOrder{
				TokenID: intent.Token,
				Price:   intent.Price,
				Size:    intent.Size,
				Side:    side,
			}, orders.CreateOrderOptions{
				SignatureType: &signatureType,
			})
			if err != nil {
				log.Printf("[Worker %d] 创建订单数据失败: %v", workerID, err)
				return
			}
			// 发送订单
			result, err := s.pmClient.PostOrder(order, intent.OrderType, false)
			if err != nil {
				if s.onReject != nil {
					s.onReject(intent, err)
				}
				log.Printf("[Worker %d] 下单失败: %v", workerID, err)
				return
			}
			if s.onSubmit != nil {
				orderID := result.Get("orderID").String()
				s.onSubmit(intent, orderID)
			}
		} else {
			var os []orders.PostOrdersArgs
			for _, intent := range intents {
				side := pmModel.BUY
				if intent.Side == model.SideSell {
					side = pmModel.SELL
				}
				signatureType := pmModel.POLY_GNOSIS_SAFE
				order, err := s.pmClient.CreateOrder(&orders.UserOrder{
					TokenID: intent.Token,
					Price:   intent.Price,
					Size:    intent.Size,
					Side:    side,
				}, orders.CreateOrderOptions{
					SignatureType: &signatureType,
				})
				if err != nil {
					log.Printf("[Worker %d] 创建订单数据失败: %v", workerID, err)
					return
				}
				os = append(os, orders.PostOrdersArgs{
					Order:     order,
					OrderType: intent.OrderType,
				})
			}
			// 发送订单
			log.Printf("[Worker %d] 下单时间: %d", workerID, time.Now().UnixMilli())
			results, err := s.pmClient.PostOrders(os, false)
			log.Printf("[Worker %d] 下单返回: %d", workerID, time.Now().UnixMilli())
			if err != nil {
				if s.onReject != nil {
					for _, intent := range intents {
						s.onReject(intent, err)
					}
				}
				log.Printf("[Worker %d] 下单失败: %v", workerID, err)
				return
			}
			arr := results.Array()
			for i, item := range arr {
				index := count - 1 - i
				intent := intents[index]
				if success := item.Get("success").Bool(); success {
					errorMsg := item.Get("errorMsg").String()
					if errorMsg != "" {
						log.Printf("[Worker %d] 下单失败: %v", workerID, errorMsg)
						if s.onReject != nil {
							s.onReject(intent, errors.New(errorMsg))
						}
					} else {
						orderID := item.Get("orderID").String()
						if s.onSubmit != nil {
							s.onSubmit(intent, orderID)
						}
					}
				} else {
					errorMsg := item.Get("errorMsg").String()
					log.Printf("[Worker %d] 下单失败: %v", workerID, errorMsg)
					if s.onReject != nil {
						s.onReject(intent, errors.New(errorMsg))
					}
				}
			}
		}
	}
}
